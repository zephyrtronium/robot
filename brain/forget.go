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
)

// ClearMsg unlearns a single message from history by Twitch message ID.
func (b *Brain) ClearMsg(ctx context.Context, msgid string) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	row := tx.StmtContext(ctx, b.stmts.historyID).QueryRowContext(ctx, msgid)
	var id int64
	var tag, msg string
	if err := row.Scan(&id, &tag, &msg); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			tx.Commit()
			return nil
		}
		tx.Rollback()
		return err
	}
	if _, err := tx.StmtContext(ctx, b.stmts.expunge).ExecContext(ctx, id); err != nil {
		tx.Rollback()
		return err
	}
	if err := b.forget(ctx, tx.StmtContext(ctx, b.stmts.forget), tag, msg); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ClearChat unlearns all recent messages from a given user in a channel.
func (b *Brain) ClearChat(ctx context.Context, channel, user string) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	expunge := tx.StmtContext(ctx, b.stmts.expunge)
	forget := tx.StmtContext(ctx, b.stmts.forget)
	rows, err := tx.StmtContext(ctx, b.stmts.historyName).QueryContext(ctx, channel, user)
	if err != nil {
		tx.Rollback()
		return err
	}
	for rows.Next() {
		var id int64
		var tag, msg string
		if err := rows.Scan(&id, &tag, &msg); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := expunge.ExecContext(ctx, id); err != nil {
			tx.Rollback()
			return err
		}
		if err := b.forget(ctx, forget, tag, msg); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// ClearPattern unlearns all messages in a channel matching a given pattern and
// returns the number of messages deleted.
func (b *Brain) ClearPattern(ctx context.Context, channel, pattern string) (int64, error) {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("couldn't open transaction: %w", err)
	}
	expunge := tx.StmtContext(ctx, b.stmts.expunge)
	forget := tx.StmtContext(ctx, b.stmts.forget)
	rows, err := tx.StmtContext(ctx, b.stmts.historyPattern).QueryContext(ctx, channel, pattern)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("couldn't get matching messages: %w", err)
	}
	var n int64
	for rows.Next() {
		var id int64
		var tag, msg string
		if err := rows.Scan(&id, &tag, &msg); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("couldn't scan results: %w", err)
		}
		res, err := expunge.ExecContext(ctx, id)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("couldn't remove from history: %w", err)
		}
		r, _ := res.RowsAffected() // This is not an error we care about.
		n += r
		if err := b.forget(ctx, forget, tag, msg); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("couldn't remove chains: %w", err)
		}
	}
	if rows.Err() != nil {
		tx.Rollback()
		return 0, rows.Err()
	}
	return n, tx.Commit()
}

func (b *Brain) forget(ctx context.Context, s *sql.Stmt, tag, msg string) error {
	// This is essentially the same as Learn, just using a different statement.
	toks := Tokens(msg)
	args := make([]interface{}, b.order+2)
	args[0] = tag
	for _, tok := range toks {
		copy(args[1:], args[2:])
		args[len(args)-1] = tok
		if _, err := s.ExecContext(ctx, args...); err != nil {
			return err
		}
		args[len(args)-1] = strings.ToLower(tok)
	}
	copy(args[1:], args[2:])
	args[len(args)-1] = nil
	if _, err := s.ExecContext(ctx, args...); err != nil {
		return err
	}
	return nil
}
