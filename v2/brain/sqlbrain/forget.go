package sqlbrain

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gitlab.com/zephyrtronium/sq"

	"github.com/zephyrtronium/robot/v2/brain"
	"github.com/zephyrtronium/robot/v2/brain/userhash"
)

// Forget deletes tuples from the database. To ensure consistency and accuracy,
// the ForgetMessage, ForgetDuring, and ForgetUserSince methods should be
// preferred where possible.
func (br *Brain) Forget(ctx context.Context, tag string, tuples []brain.Tuple) error {
	names := make([]sq.NamedArg, 2+br.order)
	names[0] = sql.Named("tag", tag)
	terms := make([]sq.NullString, 1+br.order)
	for i := 0; i < br.order; i++ {
		names[i+1] = sql.Named("p"+strconv.Itoa(i), &terms[i])
	}
	names[br.order+1] = sql.Named("suffix", &terms[br.order])
	p := make([]any, len(names))
	for i := range names {
		p[i] = names[i]
	}
	tx, err := br.db.Begin(ctx, nil)
	if err != nil {
		return fmt.Errorf("couldn't open transaction: %w", err)
	}
	defer tx.Rollback()
	for _, tup := range tuples {
		// Note that each p[i] is a named arg, and those for the prefix and
		// suffix each point to an element of terms. So, updating terms is
		// sufficient to update the query parameters.
		for i, w := range tup.Prefix {
			terms[i] = sq.NullString{String: w, Valid: w != ""}
		}
		terms[br.order] = sq.NullString{String: tup.Suffix, Valid: tup.Suffix != ""}
		// Execute the statements in order. We do this manually because the
		// arguments differ for some statements, and the SQLite3 driver
		// complains if we give the wrong ones.
		snd := func(_ sq.Result, err error) error { return err }
		steps := []func() error{
			func() error { return snd(tx.Exec(ctx, br.stmts.deleteTuple[0])) },
			func() error { return snd(tx.Exec(ctx, br.stmts.deleteTuple[1], p...)) },
			func() error { return snd(tx.Exec(ctx, br.stmts.deleteTuple[2], p[1:]...)) },
			func() error { return snd(tx.Exec(ctx, br.stmts.deleteTuple[3])) },
			func() error { return snd(tx.Exec(ctx, br.stmts.deleteTuple[4])) },
		}
		for i, step := range steps {
			err := step()
			if err != nil {
				return fmt.Errorf("couldn't remove tuples (step %d failed): %w", i, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("couldn't commit delete ops: %w", err)
	}
	return nil
}

// ForgetMessage removes tuples associated with a message from the database.
// The delete reason is set to "CLEARMSG".
func (br *Brain) ForgetMessage(ctx context.Context, msg uuid.UUID) error {
	res, err := br.db.Exec(ctx, `UPDATE Message SET deleted='CLEARMSG' WHERE id = ?`, msg)
	if err != nil {
		return fmt.Errorf("couldn't delete message %v: %w", msg, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		// Since the query succeeded, an error here is probably from the driver
		// not supporting RowsAffected (although those we use do). Don't care.
		return nil
	}
	if n == 0 {
		return fmt.Errorf("no message with id %v", msg)
	}
	return nil
}

// ForgetDuring removes tuples associated with messages learned in the given
// time span. The delete reason is set to "TIMED".
func (br *Brain) ForgetDuring(ctx context.Context, tag string, since, before time.Time) error {
	a, b := since.UnixMilli(), before.UnixMilli()
	_, err := br.db.Exec(ctx, `UPDATE Message SET deleted='TIMED' WHERE tag = ? AND time BETWEEN ? AND ?`, tag, a, b)
	if err != nil {
		return fmt.Errorf("couldn't delete messages between %v and %v: %w", since, before, err)
	}
	return nil
}

// ForgetUserSince removes tuples learned from the given user hash since a
// given time. The delete reason is set to "CLEARCHAT".
func (br *Brain) ForgetUserSince(ctx context.Context, user *userhash.Hash, since time.Time) error {
	_, err := br.db.Exec(ctx, `UPDATE Message SET deleted='CLEARCHAT' WHERE user = ? AND time >= ?`, user[:], since.UnixMilli())
	if err != nil {
		return fmt.Errorf("couldn't forget messages from %x since %v: %w", user, since, err)
	}
	return nil
}

func (br *Brain) initDelete(ctx context.Context) {
	tp, err := br.tpl.Parse(deleteTuple)
	if err != nil {
		panic(fmt.Errorf("couldn't parse tuple.delete.sql: %w", err))
	}
	const numTemplates = 5
	br.stmts.deleteTuple = make([]string, numTemplates)
	data := struct {
		Iter []struct{}
	}{
		Iter: make([]struct{}, br.order),
	}
	var b strings.Builder
	for i := range br.stmts.deleteTuple {
		b.Reset()
		err := tp.ExecuteTemplate(&b, fmt.Sprintf("tuple.delete.%d", i), &data)
		if err != nil {
			panic(fmt.Errorf("couldn't exec tuple.delete.%d: %w", i, err))
		}
		br.stmts.deleteTuple[i] = b.String()
	}
}

//go:embed templates/tuple.delete.sql
var deleteTuple string
