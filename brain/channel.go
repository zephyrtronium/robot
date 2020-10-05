package brain

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type chancfg struct {
	mu      sync.Mutex
	learn   sql.NullString
	send    sql.NullString
	lim     int
	prob    float64
	rate    *rate.Limiter
	block   *regexp.Regexp
	respond bool
	silence time.Time
	privs   map[string]string
}

// Channels returns a list of all channels configured in the database.
func (b *Brain) Channels() []string {
	b.cmu.Lock()
	defer b.cmu.Unlock()
	r := make([]string, 0, len(b.cfgs))
	for c := range b.cfgs {
		r = append(r, c)
	}
	return r
}

// Join creates a new entry for a channel if one does not already exist. If
// learn or send are nonempty, then they are used as the learn or send tag,
// respectively, updating the existing values if the channel is already known.
func (b *Brain) Join(ctx context.Context, channel, learn, send string) error {
	channel = strings.ToLower(channel)
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO chans(name) VALUES (?)`, channel); err != nil {
		tx.Rollback()
		return err
	}
	if learn != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE chans SET learn = ? WHERE name = ?`, learn, channel); err != nil {
			tx.Rollback()
			return err
		}
	}
	if send != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE chans SET send = ? WHERE name = ?`, send, channel); err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return b.Update(ctx, channel)
}

// SetPriv sets a new privilege level for a user.
func (b *Brain) SetPriv(ctx context.Context, user, channel, priv string) error {
	if b.config(channel) == nil {
		return fmt.Errorf("SetPriv: no such channel: %s", channel)
	}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	ch := sql.NullString{String: channel, Valid: channel != ""}
	switch priv {
	case "":
		_, err = tx.ExecContext(ctx, `DELETE FROM privs WHERE user=? AND chan IS ?`, user, ch)
	case "owner", "admin", "bot", "ignore":
		_, err = tx.ExecContext(ctx, `INSERT OR REPLACE INTO privs(user, chan, priv) VALUES (?, ?, ?)`, user, ch, priv)
	default:
		tx.Rollback()
		return fmt.Errorf("SetPriv: bad privilege level %q", priv)
	}
	if err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	b.cmu.Lock()
	defer b.cmu.Unlock()
	return b.hupPrivs(ctx)
}

// Silence sets a silent time on a channel. If until is the zero time, then
// null is used instead.
func (b *Brain) Silence(ctx context.Context, channel string, until time.Time) error {
	cfg := b.config(channel)
	if cfg == nil {
		return fmt.Errorf("Silence: no such channel: %q", channel)
	}
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	u := sql.NullTime{Time: until, Valid: !until.IsZero()}
	_, err := b.db.ExecContext(ctx, `UPDATE chans SET silence=? WHERE name=?`, u, channel)
	if err != nil {
		return err
	}
	cfg.silence = until
	return nil
}

// Activity sets the random response rate to a given function of its current
// response rate. If f returns a value not in the interval [0, 1], an error is
// returned. Otherwise, the new probability is returned.
func (b *Brain) Activity(ctx context.Context, channel string, f func(float64) float64) (float64, error) {
	cfg := b.config(channel)
	if cfg == nil {
		return 0, fmt.Errorf("Activity: no such channel: %q", channel)
	}
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	p := f(cfg.prob)
	// The condition is written weirdly here so that we also reject NaN.
	if !(0 <= p && p <= 1) {
		return 0, fmt.Errorf("Activity: bad probability: %g", p)
	}
	_, err := b.db.ExecContext(ctx, `UPDATE chans SET prob=? WHERE name=?`, p, channel)
	if err != nil {
		return 0, err
	}
	cfg.prob = p
	return p, nil
}

func (b *Brain) config(channel string) *chancfg {
	b.cmu.Lock()
	defer b.cmu.Unlock()
	return b.cfgs[strings.ToLower(channel)]
}

// UpdateAll fetches configuration from the database for all channels. This
// can be used to synchronize the brain's configuration with the database
// during run time.
func (b *Brain) UpdateAll(ctx context.Context) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Commit()
	row := tx.QueryRowContext(ctx, `SELECT block FROM config WHERE id=1`)
	var g sql.NullString
	if err := row.Scan(&g); err != nil {
		return err
	}
	if !g.Valid {
		g.String = "$^"
	}
	gblk, err := regexp.Compile(g.String)
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT * FROM chans`)
	if err != nil {
		return err
	}
	cfgs := make(map[string]*chancfg)
	for rows.Next() {
		var cfg chancfg
		var name string
		var r rate.Limit
		var burst int
		var block sql.NullString
		var silence sql.NullTime
		rows.Scan(&name, &cfg.learn, &cfg.send, &cfg.lim, &cfg.prob, &r, &burst, &block, &cfg.respond, &silence)
		cfg.rate = rate.NewLimiter(r, burst)
		if block.Valid {
			cfg.block, err = regexp.Compile(g.String + "|" + block.String)
			if err != nil {
				return err
			}
		} else {
			cfg.block = gblk
		}
		if silence.Valid {
			cfg.silence = silence.Time
		}
		cfgs[name] = &cfg
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	b.cmu.Lock()
	defer b.cmu.Unlock()
	b.cfgs = cfgs
	return b.hupPrivs(ctx)
}

// hupPrivs updates per-channel user privileges and emotes. The config mutex
// must be held when calling this method.
func (b *Brain) hupPrivs(ctx context.Context) error {
	emotes := make(map[string]emopt)
	for name, cfg := range b.cfgs {
		rows, err := b.db.QueryContext(ctx, `SELECT user, priv FROM privs WHERE chan=? OR chan IS NULL ORDER BY chan NULLS FIRST`, name)
		if err != nil {
			return err
		}
		privs := make(map[string]string)
		for rows.Next() {
			var user, priv string
			if err := rows.Scan(&user, &priv); err != nil {
				return err
			}
			privs[user] = priv
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		cfg.mu.Lock()
		tag := cfg.send
		cfg.mu.Unlock()
		rows, err = b.db.QueryContext(ctx, `SELECT emote, weight FROM emotes WHERE tag=? OR tag IS NULL`, tag)
		if err != nil {
			return err
		}
		var sum int64
		var opts []optfreq
		for rows.Next() {
			var opt optfreq
			if err := rows.Scan(&opt.w, &opt.n); err != nil {
				return err
			}
			opt.n, sum = opt.n+sum, opt.n+sum
			opts = append(opts, opt)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		cfg.mu.Lock()
		cfg.privs = privs
		cfg.mu.Unlock()
		if tag.Valid {
			emotes[tag.String] = emopt{s: sum, e: opts}
		}
	}
	b.emu.Lock()
	b.emotes = emotes
	b.emu.Unlock()
	return nil
}

// Update updates config for a single channel, including user privileges and
// emotes (the latter for all channels which share a send tag with channel).
func (b *Brain) Update(ctx context.Context, channel string) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Commit()
	row := tx.QueryRowContext(ctx, `SELECT learn, send, lim, prob, rate, burst, block, respond, silence FROM chans WHERE name=?`, channel)
	var cfg chancfg
	var r rate.Limit
	var burst int
	var block sql.NullString
	var silence sql.NullTime
	if err := row.Scan(&cfg.learn, &cfg.send, &cfg.lim, &cfg.prob, &r, &burst, &block, &cfg.respond, &silence); err != nil {
		return err
	}
	cfg.rate = rate.NewLimiter(r, burst)
	if block.Valid {
		cfg.block, err = regexp.Compile(block.String)
		if err != nil {
			return err
		}
	} else {
		cfg.block = regexp.MustCompile("$^")
	}
	if silence.Valid {
		cfg.silence = silence.Time
	}
	rows, err := tx.QueryContext(ctx, `SELECT user, priv FROM privs WHERE chan=? OR chan IS NULL ORDER BY chan NULLS FIRST`, channel)
	if err != nil {
		return err
	}
	cfg.privs = make(map[string]string)
	for rows.Next() {
		var user, priv string
		if err := rows.Scan(&user, &priv); err != nil {
			return err
		}
		cfg.privs[user] = priv
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	rows, err = tx.QueryContext(ctx, `SELECT emote, weight FROM emotes WHERE tag=? OR tag IS NULL`, cfg.send)
	if err != nil {
		return err
	}
	var em emopt
	for rows.Next() {
		var opt optfreq
		if err := rows.Scan(&opt.w, &opt.n); err != nil {
			return err
		}
		opt.n, em.s = opt.n+em.s, opt.n+em.s
		em.e = append(em.e, opt)
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	b.cmu.Lock()
	b.cfgs[channel] = &cfg
	b.cmu.Unlock()
	if cfg.send.Valid {
		b.emu.Lock()
		b.emotes[cfg.send.String] = em
		b.emu.Unlock()
	}
	return nil
}

// Privilege returns the privilege level associated with a user in a channel.
// If the channel is unrecognized, the privilege level is always "ignore".
func (b *Brain) Privilege(ctx context.Context, channel, nick, badges string) (string, error) {
	cfg := b.config(channel)
	if cfg == nil {
		return "ignore", nil
	}
	// Channel-specific privileges override badges.
	cfg.mu.Lock()
	priv, ok := cfg.privs[strings.ToLower(nick)]
	cfg.mu.Unlock()
	if ok {
		return priv, nil
	}
	// Check badges. Broadcasters and moderators default to admin privileges;
	// Twitch staff, admins, and global moderators default to owner
	// privileges; and all others default to none.
	for badges != "" {
		// Index rather than use Split because we don't want to allocate the
		// slices. This method is called potentially thousands of times per
		// second, so it would generate a lot of garbage.
		k := strings.IndexByte(badges, ',')
		badge := badges
		if k > 0 {
			badge = badges[:k]
			badges = badges[k+1:]
		} else {
			break
		}
		k = strings.IndexByte(badge, '/')
		// Assume k > 0, otherwise there's been a significant change to the
		// Twitch API that we should loudly know about.
		switch badge[:k] {
		case "broadcaster", "moderator":
			return "admin", nil
		case "staff", "admin", "global_mod":
			return "owner", nil
		}
	}
	return "", nil
}
