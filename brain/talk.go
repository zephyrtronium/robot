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
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"unicode/utf8"

	"github.com/zephyrtronium/robot/irc"
)

// Talk constructs a string up to length n from the database, starting with the
// given chain. chain may be any length or nil; if it is shorter than b's
// order, it is padded with nulls as needed, and if it is longer, then only the
// last (order) elements are taken (but the generated message still contains
// the earlier ones). If the resulting walk has no more words than the starting
// chain, then the result is the empty string.
func (b *Brain) Talk(ctx context.Context, tag string, chain []string, n int) string {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return ""
	}
	defer tx.Commit()
	if len(chain) > 0 {
		// Verify that the chain is plausible, so this can't cause the bot to
		// say any message.
		s := tx.StmtContext(ctx, b.stmts.verify)
		args := []interface{}{tag, nil, chain[0]}
		var r bool
		for _, w := range chain[1:] {
			args[1], args[2] = strings.ToLower(args[2].(string)), w
			if err := s.QueryRowContext(ctx, args...).Scan(&r); err != nil {
				return ""
			}
		}
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
	think := make([]*sql.Stmt, len(b.stmts.think))
	for i, s := range b.stmts.think {
		think[i] = tx.StmtContext(ctx, s)
	}
	for {
		w, err := b.think(ctx, think, args)
		if w == "" || err != nil {
			break
		}
		if l += utf8.RuneCountInString(w) + 1; l > n {
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

// Effect selects a random effect from the given tag. The result is the empty
// string if the tag is unused by any channel or if the selected effect
// corresponds to a null SQL text.
func (b *Brain) Effect(ctx context.Context, tag string) string {
	b.fmu.Lock()
	ef := b.effects[tag]
	b.fmu.Unlock()
	if ef.s <= 0 {
		return ""
	}
	x := b.intn(ef.s)
	for _, v := range ef.e {
		if x < v.n {
			if !v.w.Valid {
				return ""
			}
			return v.w.String
		}
	}
	return ""
}

// EffectIn calls Effect with the given channel's send tag.
func (b *Brain) EffectIn(ctx context.Context, channel string) string {
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
	return b.Effect(ctx, tag.String)
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

// SetEchoDir sets the directory for echoing.
func (b *Brain) SetEchoDir(dir string) {
	b.echoto.Store(dir)
}

// EchoTo returns the directory for echoing messages generated for the given
// channel. If no such directory is set, then this returns the empty string.
func (b *Brain) EchoTo(channel string) string {
	cfg := b.config(channel)
	cfg.mu.Lock()
	echo := cfg.echo
	cfg.mu.Unlock()
	if !echo {
		return ""
	}
	return b.echoto.Load().(string)
}

// doEcho echoes a message to the given directory using tag as part of the
// filename. If dir or msg is the empty string, or if an error occurs, this
// silently does nothing.
func (b *Brain) doEcho(tag, dir, msg string) {
	if dir == "" || msg == "" {
		return
	}
	f, err := ioutil.TempFile(dir, tag)
	if err != nil {
		return
	}
	f.WriteString(msg)
	f.Close()
}

// CheckCopypasta returns nil if the message is copypasta that the bot should
// repeat, considering channel settings and whether the bot has already
// copypasted it, or an error explaining why the bot should not repeat it.
func (b *Brain) CheckCopypasta(ctx context.Context, msg irc.Message) error {
	if msg.Command != "PRIVMSG" {
		return fmt.Errorf("message not PRIVMSG")
	}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to open transaction: %w", err)
	}
	defer tx.Commit()
	var ok sql.NullBool
	res := tx.StmtContext(ctx, b.stmts.memeDetector).QueryRowContext(ctx, msg.To(), msg.Trailing)
	if err := res.Scan(&ok); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NoCopypasta
		}
		return fmt.Errorf("error checking message for copypasta: %w", err)
	}
	if !ok.Bool {
		return NoCopypasta
	}
	var n int64
	res = tx.StmtContext(ctx, b.stmts.familiar).QueryRowContext(ctx, nil, msg.Trailing)
	if err := res.Scan(&n); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("error checking generated messages for copypasta: %w", err)
	}
	if n > 0 {
		return NoCopypasta
	}
	if _, err := tx.StmtContext(ctx, b.stmts.copypasta).ExecContext(ctx, msg.Time, msg.To(), msg.Trailing); err != nil {
		return fmt.Errorf("error reserving copypasta: %w", err)
	}
	return nil
}

type noCopypasta struct{}

func (noCopypasta) Error() string {
	return "not copypasta"
}

// NoCopypasta is a sentinel error returned by CheckCopypasta to indicate that
// a message is not a meme for a generic reason.
var NoCopypasta = noCopypasta{}
