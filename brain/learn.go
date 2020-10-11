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
	"strings"

	"github.com/zephyrtronium/robot/irc"
)

// Learn adds a message to the history and its chains to the tuples database.
// If the channel to which it was sent is configured to ignore the message for
// any reason, then this is a no-op.
func (b *Brain) Learn(ctx context.Context, msg irc.Message) error {
	channel := strings.ToLower(msg.To())
	cfg := b.config(channel)
	if cfg == nil {
		// unrecognized channel
		return fmt.Errorf("Learn: no such channel: %s", channel)
	}
	if b.ignoremsg(ctx, cfg, msg) {
		return nil
	}
	cfg.mu.Lock()
	tag := cfg.learn.String
	cfg.mu.Unlock()
	toks := Tokens(msg.Trailing)
	if len(toks) == 0 {
		return nil
	}
	args := make([]interface{}, b.order+2)
	args[0] = tag
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Learn: error opening tx: %w", err)
	}
	s := tx.StmtContext(ctx, b.stmts.learn)
	for _, tok := range toks {
		copy(args[1:], args[2:])
		args[len(args)-1] = tok
		if _, err := s.ExecContext(ctx, args...); err != nil {
			tx.Rollback()
			return fmt.Errorf("Learn: error learning %+v: %w", args, err)
		}
		// While we still have the token easily available, make it lowercase.
		// On the next iteration, or after this loop if this is the last one,
		// it will be copied into the prefix tuple.
		args[len(args)-1] = strings.ToLower(tok)
	}
	// Add a final tuple for the end of walk.
	copy(args[1:], args[2:])
	args[len(args)-1] = nil
	if _, err := s.ExecContext(ctx, args...); err != nil {
		tx.Rollback()
		return fmt.Errorf("Learn: error learning end-of-message %+v: %w", args, err)
	}
	// Add the message to history.
	id, _ := msg.Tag("id")
	if _, err := tx.StmtContext(ctx, b.stmts.record).ExecContext(ctx, id, msg.Time, msg.Nick, channel, tag, msg.Trailing); err != nil {
		tx.Rollback()
		return fmt.Errorf("Learn: error recording message: %w", err)
	}
	return tx.Commit()
}

// LearnTuple directly learns a single chain. Panics if len(pre) is not exactly
// the chain order. It exists to facilitate converting ancient chain formats to
// the modern one.
func (b *Brain) LearnTuple(ctx context.Context, tag string, pre []sql.NullString, suf sql.NullString) (sql.Result, error) {
	if len(pre) != b.order {
		panic(fmt.Errorf("brain: mismatched tuple length %d and order %d", len(pre), b.order))
	}
	args := make([]interface{}, b.order+2)
	args[0] = tag
	for i, v := range pre {
		args[i+1] = v
	}
	args[b.order+1] = suf
	return b.stmts.learn.ExecContext(ctx, args...)
}

// ignoremsg returns whether the given message should not be learned.
func (b *Brain) ignoremsg(ctx context.Context, cfg *chancfg, msg irc.Message) bool {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	if !cfg.learn.Valid {
		return true
	}
	if msg.Time.Before(cfg.silence) {
		return true
	}
	switch cfg.privs[strings.ToLower(msg.Nick)] {
	case "ignore", "bot", "privacy":
		return true
	case "", "admin", "owner": // do nothing
	default: // TODO: complain
	}
	if priv := cfg.privs[strings.ToLower(msg.Nick)]; priv == "ignore" || priv == "bot" {
		return true
	}
	if cfg.block.MatchString(msg.Trailing) {
		return true
	}
	row := b.stmts.familiar.QueryRowContext(ctx, cfg.send, msg.Trailing)
	var x int
	if err := row.Scan(&x); err != nil {
		return !errors.Is(err, sql.ErrNoRows)
	}
	return false
}
