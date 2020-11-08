/*
Copyright (C) 2020  Branden J Brown

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package brain

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	echo    bool
	online  bool
	privs   map[string]string

	wait atomic.Value // *rate.Limiter, not guarded by mu
}

// Channels returns a list of all channels configured in the database.
func (b *Brain) Channels() []string {
	b.cmu.Lock()
	defer b.cmu.Unlock()
	r := make([]string, 0, len(b.cfgs))
	for c := range b.cfgs {
		r = append(r, c)
	}
	sort.Strings(r)
	return r
}

// Join creates a new entry for a channel if one does not already exist. If
// learn or send are nonempty, then they are used as the learn or send tag,
// respectively, updating the existing values if the channel is already known.
func (b *Brain) Join(ctx context.Context, channel, learn, send string) error {
	channel = strings.ToLower(channel)
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error opening join transaction: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO chans(name) VALUES (?)`, channel); err != nil {
		tx.Rollback()
		return fmt.Errorf("error inserting new channel: %w", err)
	}
	if learn != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE chans SET learn = ? WHERE name = ?`, learn, channel); err != nil {
			tx.Rollback()
			return fmt.Errorf("error setting learn tag: %w", err)
		}
	}
	if send != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE chans SET send = ? WHERE name = ?`, send, channel); err != nil {
			tx.Rollback()
			return fmt.Errorf("error setting send tag: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing join changes: %w", err)
	}
	return b.Update(ctx, channel)
}

// SetPriv sets a new privilege level for a user.
func (b *Brain) SetPriv(ctx context.Context, user, channel, priv string) error {
	if channel != "" && b.config(channel) == nil {
		return fmt.Errorf("no such channel: %s", channel)
	}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error opening privileges transaction: %w", err)
	}
	ch := sql.NullString{String: channel, Valid: channel != ""}
	switch priv {
	case "":
		_, err = tx.ExecContext(ctx, `DELETE FROM privs WHERE user=? AND chan IS ?`, user, ch)
	case "owner", "admin", "bot", "privacy", "ignore":
		_, err = tx.ExecContext(ctx, `INSERT OR REPLACE INTO privs(user, chan, priv) VALUES (?, ?, ?)`, user, ch, priv)
	default:
		tx.Rollback()
		return fmt.Errorf("bad privilege level %q", priv)
	}
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error setting privileges: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing privileges changes: %w", err)
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
		return fmt.Errorf("no such channel: %q", channel)
	}
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	u := sql.NullTime{Time: until, Valid: !until.IsZero()}
	_, err := b.db.ExecContext(ctx, `UPDATE chans SET silence=? WHERE name=?`, u, channel)
	if err != nil {
		return fmt.Errorf("error updating silence: %w", err)
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
		return 0, fmt.Errorf("no such channel: %q", channel)
	}
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	p := f(cfg.prob)
	// The condition is written weirdly here so that we also reject NaN.
	if !(0 <= p && p <= 1) {
		return 0, fmt.Errorf("bad probability: %g", p)
	}
	_, err := b.db.ExecContext(ctx, `UPDATE chans SET prob=? WHERE name=?`, p, channel)
	if err != nil {
		return 0, fmt.Errorf("error updating probability to %g: %w", p, err)
	}
	cfg.prob = p
	return p, nil
}

// SetOnline marks a channel as online to enable learning or offline to disable
// it. All channels are offline until this method is used to set them to
// online. If the channel is not found, this has no effect.
func (b *Brain) SetOnline(channel string, online bool) {
	cfg := b.config(channel)
	if cfg == nil {
		return
	}
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.online = online
}

// SetWait sets the hard rate limiter for a channel. If the new rate equals the
// old one, or if the channel is not found, this has no effect. If SetWait
// would set a new rate limit, it first waits for the old one.
func (b *Brain) SetWait(ctx context.Context, channel string, limit rate.Limit) {
	cfg := b.config(channel)
	if cfg == nil {
		return
	}
	w := cfg.wait.Load().(*rate.Limiter)
	w.SetLimit(limit)
}

// Wait waits for the channel-specific rate limit, or for the global one if
// there is no channel-specific one.
func (b *Brain) Wait(ctx context.Context, channel string) {
	cfg := b.config(channel)
	if cfg != nil {
		r := cfg.wait.Load().(*rate.Limiter)
		r.Wait(ctx)
		return
	}
	now := time.Now()
	r0 := b.wait[0].ReserveN(now, 1)
	r1 := b.wait[1].ReserveN(now, 1)
	var d time.Duration
	switch {
	case r0.OK() && r1.OK():
		d0, d1 := r0.DelayFrom(now), r1.DelayFrom(now)
		if d0 <= d1 {
			d = d0
		} else {
			d = d1
		}
	case r0.OK():
		d = r0.DelayFrom(now)
	case r1.OK():
		d = r1.DelayFrom(now)
	default:
		return
	}
	if d == 0 {
		// Skip creating garbage if we don't need to wait.
		return
	}
	select {
	case <-ctx.Done(): // do nothing
	case <-time.After(d): // do nothing
	}
}

// SendTag gets the send tag associated with the given channel. The second
// returned value is false when the channel has no tag.
func (b *Brain) SendTag(channel string) (string, bool) {
	cfg := b.config(channel)
	if cfg == nil {
		return "", false
	}
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.send.String, cfg.send.Valid
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
		return fmt.Errorf("error opening update transaction: %w", err)
	}
	defer tx.Commit()
	row := tx.QueryRowContext(ctx, `SELECT block FROM config WHERE id=1`)
	var g sql.NullString
	if err := row.Scan(&g); err != nil {
		return fmt.Errorf("error reading global block: %w", err)
	}
	if !g.Valid {
		g.String = "$^"
	}
	gblk, err := regexp.Compile(g.String)
	if err != nil {
		return fmt.Errorf("error compiling global regexp %q: %w", g.String, err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT * FROM chans`)
	if err != nil {
		return fmt.Errorf("error getting channel info: %w", err)
	}
	cfgs := make(map[string]*chancfg)
	for rows.Next() {
		var cfg chancfg
		var name string
		var r rate.Limit
		var burst int
		var block sql.NullString
		var silence sql.NullTime
		rows.Scan(&name, &cfg.learn, &cfg.send, &cfg.lim, &cfg.prob, &r, &burst, &block, &cfg.respond, &silence, &cfg.echo)
		cfg.rate = rate.NewLimiter(r, burst)
		if block.Valid {
			re := g.String + "|" + block.String
			cfg.block, err = regexp.Compile(re)
			if err != nil {
				return fmt.Errorf("error compiling regexp (%s) for %s: %w", re, name, err)
			}
		} else {
			cfg.block = gblk
		}
		if silence.Valid {
			cfg.silence = silence.Time
		}
		if old := b.config(name); old != nil {
			// Copy the old rate limiter.
			cfg.wait.Store(old.wait.Load())
		} else {
			// Default rate limit averages to 20 messages per 30 seconds, per
			// Twitch documentation: https://dev.twitch.tv/docs/irc/guide
			cfg.wait.Store(rate.NewLimiter(20/30.0, 1))
		}
		cfgs[name] = &cfg
	}
	if rows.Err() != nil {
		return fmt.Errorf("error getting channel info: %w", rows.Err())
	}
	b.cmu.Lock()
	defer b.cmu.Unlock()
	b.cfgs = cfgs
	return b.hupPrivs(ctx)
}

// hupPrivs updates per-channel user privileges, emotes, and effects. The
// config mutex must be held when calling this method.
func (b *Brain) hupPrivs(ctx context.Context) error {
	emotes := make(map[string]emopt)
	effects := make(map[string]emopt)
	for name, cfg := range b.cfgs {
		rows, err := b.db.QueryContext(ctx, `SELECT user, priv FROM privs WHERE chan=? OR chan IS NULL ORDER BY chan NULLS FIRST`, name)
		if err != nil {
			return fmt.Errorf("error getting privileges for %s: %w", name, err)
		}
		privs := make(map[string]string)
		for rows.Next() {
			var user, priv string
			if err := rows.Scan(&user, &priv); err != nil {
				return fmt.Errorf("error reading privileges for %s: %w", name, err)
			}
			privs[user] = priv
		}
		if rows.Err() != nil {
			return fmt.Errorf("error getting privileges for %s: %w", name, err)
		}
		cfg.mu.Lock()
		cfg.privs = privs
		tag := cfg.send
		cfg.mu.Unlock()
		if _, ok := emotes[tag.String]; !ok {
			rows, err = b.db.QueryContext(ctx, `SELECT emote, SUM(weight) FROM emotes WHERE tag=? OR tag IS NULL GROUP BY emote`, tag)
			if err != nil {
				return fmt.Errorf("error getting emotes for %s: %w", name, err)
			}
			var sum int64
			var opts []optfreq
			for rows.Next() {
				var opt optfreq
				if err := rows.Scan(&opt.w, &opt.n); err != nil {
					return fmt.Errorf("error reading emotes for %s: %w", name, err)
				}
				if opt.n <= 0 {
					continue
				}
				opt.n, sum = opt.n+sum, opt.n+sum
				opts = append(opts, opt)
			}
			if rows.Err() != nil {
				return fmt.Errorf("error getting emotes for %s: %w", name, rows.Err())
			}
			if tag.Valid {
				emotes[tag.String] = emopt{s: sum, e: opts}
			}
			rows, err = b.db.QueryContext(ctx, `SELECT effect, SUM(weight) FROM effects WHERE tag=? OR tag IS NULL GROUP BY effect`, tag)
			if err != nil {
				return fmt.Errorf("error getting effects for %s: %w", name, err)
			}
			sum, opts = 0, nil
			for rows.Next() {
				var opt optfreq
				if err := rows.Scan(&opt.w, &opt.n); err != nil {
					return fmt.Errorf("error reading effects for %s: %w", name, err)
				}
				if opt.n <= 0 {
					continue
				}
				opt.n, sum = opt.n+sum, opt.n+sum
				opts = append(opts, opt)
			}
			if rows.Err() != nil {
				return fmt.Errorf("error getting effects for %s: %w", name, rows.Err())
			}
			if tag.Valid {
				effects[tag.String] = emopt{s: sum, e: opts}
			}
		}
	}
	b.emu.Lock()
	b.emotes = emotes
	b.emu.Unlock()
	b.fmu.Lock()
	b.effects = effects
	b.fmu.Unlock()
	return nil
}

// Update updates config for a single channel, including user privileges and
// emotes (the latter for all channels which share a send tag with channel).
func (b *Brain) Update(ctx context.Context, channel string) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error opening transaction: %w", err)
	}
	defer tx.Commit()
	row := tx.QueryRowContext(ctx, `SELECT learn, send, lim, prob, rate, burst, chans.block, respond, silence, echo, config.block FROM chans, config WHERE chans.name=? AND config.id=1`, channel)
	var cfg chancfg
	var r rate.Limit
	var burst int
	var block, gblock sql.NullString
	var silence sql.NullTime
	if err := row.Scan(&cfg.learn, &cfg.send, &cfg.lim, &cfg.prob, &r, &burst, &block, &cfg.respond, &silence, &cfg.echo, &gblock); err != nil {
		return fmt.Errorf("error reading config for %s: %w", channel, err)
	}
	cfg.rate = rate.NewLimiter(r, burst)
	re := gblock.String
	if !gblock.Valid {
		re = "$^"
	}
	if block.Valid {
		re += "|" + block.String
	}
	cfg.block, err = regexp.Compile(re)
	if err != nil {
		return fmt.Errorf("error compiling regular expression (%s): %w", re, err)
	}
	if silence.Valid {
		cfg.silence = silence.Time
	}
	if old := b.config(channel); old != nil {
		cfg.wait.Store(old.wait.Load())
	} else {
		cfg.wait.Store(rate.NewLimiter(20/30.0, 1))
	}
	rows, err := tx.QueryContext(ctx, `SELECT user, priv FROM privs WHERE chan=? OR chan IS NULL ORDER BY chan NULLS FIRST`, channel)
	if err != nil {
		return fmt.Errorf("error getting privileges: %w", err)
	}
	cfg.privs = make(map[string]string)
	for rows.Next() {
		var user, priv string
		if err := rows.Scan(&user, &priv); err != nil {
			return fmt.Errorf("error reading privileges: %w", err)
		}
		cfg.privs[user] = priv
	}
	if rows.Err() != nil {
		return fmt.Errorf("error getting privileges: %w", rows.Err())
	}
	rows, err = tx.QueryContext(ctx, `SELECT emote, SUM(weight) FROM emotes WHERE tag=? OR tag IS NULL GROUP BY emote`, cfg.send)
	if err != nil {
		return fmt.Errorf("error getting emotes: %w", err)
	}
	var em emopt
	for rows.Next() {
		var opt optfreq
		if err := rows.Scan(&opt.w, &opt.n); err != nil {
			return fmt.Errorf("error reading emotes: %w", err)
		}
		opt.n, em.s = opt.n+em.s, opt.n+em.s
		em.e = append(em.e, opt)
	}
	if rows.Err() != nil {
		return fmt.Errorf("error getting emotes: %w", rows.Err())
	}
	rows, err = tx.QueryContext(ctx, `SELECT effect, SUM(weight) FROM effects WHERE tag=? OR tag IS NULL GROUP BY effect`, cfg.send)
	if err != nil {
		return fmt.Errorf("error getting effects: %w", err)
	}
	var ef emopt
	for rows.Next() {
		var opt optfreq
		if err := rows.Scan(&opt.w, &opt.n); err != nil {
			return fmt.Errorf("error reading effects: %w", err)
		}
		opt.n, ef.s = opt.n+ef.s, opt.n+ef.s
		ef.e = append(ef.e, opt)
	}
	if rows.Err() != nil {
		return fmt.Errorf("error getting effects: %w", rows.Err())
	}
	b.cmu.Lock()
	b.cfgs[channel] = &cfg
	b.cmu.Unlock()
	if cfg.send.Valid {
		b.emu.Lock()
		b.emotes[cfg.send.String] = em
		b.emu.Unlock()
		b.fmu.Lock()
		b.effects[cfg.send.String] = ef
		b.fmu.Unlock()
	}
	return nil
}

// Privilege returns the privilege level associated with a user in a channel.
// If the channel is unrecognized, the privilege level is always "ignore".
func (b *Brain) Privilege(ctx context.Context, channel, nick string, badges []string) (string, error) {
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
	for _, badge := range badges {
		switch badge {
		case "broadcaster", "moderator":
			return "admin", nil
		case "staff", "admin", "global_mod":
			return "owner", nil
		}
	}
	return "", nil
}

// Debug returns strings describing the current status of a channel. If the
// channel name is unknown, the results are the empty string.
func (b *Brain) Debug(channel string) (status, block, privs string) {
	cfg := b.config(channel)
	if cfg == nil {
		return
	}
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	learn := "<null>"
	if cfg.learn.Valid {
		learn = strconv.Quote(cfg.learn.String)
	}
	send := "<null>"
	if cfg.send.Valid {
		send = strconv.Quote(cfg.send.String)
	}
	silence := "<never>"
	if !cfg.silence.IsZero() {
		silence = cfg.silence.Format(time.Stamp)
	}
	w := cfg.wait.Load().(*rate.Limiter)
	status = fmt.Sprintf("name=%s learn=%s send=%s lim=%d prob=%g rate=%g burst=%d respond=%t silence=%s online=%t hard-rate=%g", channel, learn, send, cfg.lim, cfg.prob, cfg.rate.Limit(), cfg.rate.Burst(), cfg.respond, silence, cfg.online, w.Limit())
	block = cfg.block.String()
	privs = fmt.Sprint(cfg.privs)
	return status, block, privs
}

// DebugTag returns lists of emotes and effects for a send tag. If the tag is
// not used for any channel, the results are empty.
func (b *Brain) DebugTag(tag string) (emotes, effects []string) {
	b.emu.Lock()
	em := b.emotes[tag]
	b.emu.Unlock()
	b.fmu.Lock()
	ef := b.effects[tag]
	b.fmu.Unlock()
	emotes = make([]string, 0, len(em.e))
	effects = make([]string, 0, len(ef.e))
	for _, v := range em.e {
		if v.w.Valid {
			emotes = append(emotes, fmt.Sprintf("%q %d", v.w.String, v.n))
		} else {
			emotes = append(emotes, fmt.Sprintf("NULL %d", v.n))
		}
	}
	for _, v := range ef.e {
		if v.w.Valid {
			effects = append(effects, fmt.Sprintf("%q %d", v.w.String, v.n))
		} else {
			effects = append(effects, fmt.Sprintf("NULL %d", v.n))
		}
	}
	return emotes, effects
}
