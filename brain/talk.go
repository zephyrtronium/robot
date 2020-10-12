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
	"strings"

	"github.com/zephyrtronium/robot/irc"
)

// Talk constructs a string up to length n from the database, starting with the
// given chain. chain may be any length or nil; if it is shorter than b's
// order, it is padded with nulls as needed, and if it is longer, then only the
// last (order) elements are taken (but the generated message still contains
// the earlier ones). If the resulting walk has no more words than the starting
// chain, then the result is the empty string.
func (b *Brain) Talk(ctx context.Context, tag string, chain []string, n int) string {
	if chain != nil {
		// Ensure that an append causes a copy.
		chain = chain[:len(chain):len(chain)]
	}
	min := len(chain)
	args := make([]interface{}, b.order+1)
	args[0] = tag
	l := 0
	if len(chain) <= b.order {
		for i, v := range chain {
			args[1+b.order-len(chain)+i] = strings.ToLower(v)
			l += len(v) + 1
		}
	} else {
		for i, v := range chain[len(chain)-b.order:] {
			args[1+i] = strings.ToLower(v)
		}
		for _, v := range chain {
			l += len(v) + 1
		}
	}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return ""
	}
	defer tx.Commit()
	think := make([]*sql.Stmt, len(b.stmts.think))
	for i, s := range b.stmts.think {
		think[i] = tx.StmtContext(ctx, s)
	}
	for {
		w, err := b.think(ctx, think, args)
		if w == "" || err != nil {
			break
		}
		if l += len(w) + 1; l > n {
			break
		}
		chain = append(chain, w)
		copy(args[1:], args[2:])
		args[len(args)-1] = strings.ToLower(w)
	}
	if len(chain) <= min {
		return ""
	}
	msg := strings.TrimSpace(strings.Join(chain, " ") + " " + b.Emote(ctx, tag))
	tx.StmtContext(ctx, b.stmts.thought).ExecContext(ctx, tag, msg)
	return msg
}

// TalkIn calls Talk with a given channel's tag and limit settings. The result
// is an empty string if the channel does not exist or has no send tag, or if
// any other error occurs.
func (b *Brain) TalkIn(ctx context.Context, channel string, chain []string) string {
	cfg := b.config(channel)
	if cfg == nil {
		return ""
	}
	cfg.mu.Lock()
	tag := cfg.send
	n := cfg.lim
	cfg.mu.Unlock()
	if !tag.Valid {
		return ""
	}
	return b.Talk(ctx, tag.String, chain, n)
}

func (b *Brain) think(ctx context.Context, think []*sql.Stmt, args []interface{}) (string, error) {
	opts := b.opts.Get().([]optfreq)
	defer func() {
		if opts != nil {
			b.opts.Put(opts[:0])
		}
	}()
	args = append([]interface{}{}, args...) // preserve args
	for i, s := range think {
		rows, err := s.QueryContext(ctx, args...)
		if err != nil {
			return "", err
		}
		for rows.Next() {
			var w optfreq
			if err := rows.Scan(&w.w, &w.n); err != nil {
				return "", err
			}
			opts = append(opts, w)
		}
		if rows.Err() != nil {
			return "", rows.Err()
		}
		// If we already have enough options, we don't need to search for more.
		// TODO: use a condition that's justifiable
		if len(opts) >= b.order*(i+2) {
			break
		}
	}
	if len(opts) == 0 {
		return "", nil
	}
	var s int64
	for i, v := range opts {
		opts[i].n, s = s+v.n, s+v.n
	}
	p := b.intn(s)
	var w sql.NullString
	for _, v := range opts {
		if v.n > p {
			w = v.w
			break
		}
	}
	if !w.Valid {
		return "", nil
	}
	return w.String, nil
}

// Emote selects a random emote from the given tag. The result is the empty
// string if the tag is unused for sending by any channel or if the selected
// emote corresponds to a null SQL text.
func (b *Brain) Emote(ctx context.Context, tag string) string {
	b.emu.Lock()
	em := b.emotes[tag]
	b.emu.Unlock()
	if em.s <= 0 {
		return ""
	}
	x := b.intn(em.s)
	for _, v := range em.e {
		if x < v.n {
			if !v.w.Valid {
				return ""
			}
			return v.w.String
		}
	}
	return ""
}

// EmoteIn calls Emote with the given channel's send tag.
func (b *Brain) EmoteIn(ctx context.Context, channel string) string {
	cfg := b.config(channel)
	if cfg == nil {
		return ""
	}
	cfg.mu.Lock()
	tag := cfg.send
	cfg.mu.Unlock()
	if !tag.Valid {
		return ""
	}
	return b.Emote(ctx, tag.String)
}

// Privmsg creates an IRC PRIVMSG with a random emote appended to the message.
func (b *Brain) Privmsg(ctx context.Context, to, msg string) irc.Message {
	return irc.Message{
		Command:  "PRIVMSG",
		Params:   []string{to},
		Trailing: strings.TrimSpace(msg + " " + b.EmoteIn(ctx, to)),
	}
}

// ShouldTalk determines whether the given message should trigger talking. A
// non-nil returned error indicates the reason the message was denied. If
// random is true, then this incorporates the prob channel config setting;
// otherwise, this incorporates the respond setting.
func (b *Brain) ShouldTalk(ctx context.Context, msg irc.Message, random bool) error {
	if msg.Command != "PRIVMSG" || len(msg.Params) == 0 {
		return fmt.Errorf("not a PRIVMSG sent to a channel")
	}
	cfg := b.config(msg.To())
	if cfg == nil {
		return fmt.Errorf("unrecognized channel")
	}
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	if !cfg.send.Valid {
		return fmt.Errorf("channel has no send tag")
	}
	switch priv := cfg.privs[strings.ToLower(msg.Nick)]; priv {
	case "ignore":
		return fmt.Errorf("user is ignored")
	case "", "owner", "admin", "bot", "privacy": // do nothing
	default:
		return fmt.Errorf("unrecognized privilege level %q", priv)
	}
	if random {
		if msg.Time.Before(cfg.silence) {
			return fmt.Errorf("channel is silenced until %v", cfg.silence)
		}
		if b.unifm() > cfg.prob {
			return fmt.Errorf("rng")
		}
	} else {
		if !cfg.respond {
			return fmt.Errorf("channel has respond disabled")
		}
	}
	// Note that the rate limiter must be the last check, because it
	// consumes a resource if the check passes.
	if !cfg.rate.AllowN(msg.Time, 1) {
		return fmt.Errorf("rate limited")
	}
	return nil
}
