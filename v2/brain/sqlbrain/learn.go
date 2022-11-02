package sqlbrain

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"github.com/zephyrtronium/robot/v2/brain"
)

// Learn learns a message.
func (br *Brain) Learn(ctx context.Context, meta *brain.MessageMeta, tuples []brain.Tuple) error {
	s := br.tupleInsert(tuples)
	// Convert the tuples to SQL parameters. The first parameter to the SQL
	// statement is the message ID, and all the rest are the tuple terms in
	// sequence. Since the parameters are passed as ...any, we need to build a
	// slice of all of them. Using pointers to the strings instead of the
	// strings directly avoids extra allocations.
	p := make([]any, 1, 1+len(tuples)*br.order)
	p[0] = &meta.ID
	for i := range tuples {
		tuple := &tuples[i]
		for i := range tuple.Prefix {
			p = append(p, &tuple.Prefix[i])
		}
		p = append(p, &tuple.Suffix)
	}
	// Now execute SQL statements.
	tx, err := br.db.Begin(ctx, nil)
	if err != nil {
		return fmt.Errorf("couldn't open transaction: %w", err)
	}
	defer tx.Rollback()
	// We must insert the message first because tuples use it as an FK.
	// The INSERT returns the delete reason, which is probably but not
	// certainly NULL.
	var deleted sql.NullString
	id := &meta.ID
	user := meta.User[:]
	err = tx.QueryRow(ctx, insertMessage, id, user, meta.Tag, meta.Time.UnixMilli()).Scan(&deleted)
	if err != nil {
		return fmt.Errorf("couldn't insert message: %w", err)
	}
	if deleted.Valid {
		return fmt.Errorf("message %v was already deleted: %s", meta.ID, deleted.String)
	}
	// Now insert tuples.
	_, err = tx.Exec(ctx, s, p...)
	if err != nil {
		return fmt.Errorf("couldn't insert tuples: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("couldn't commit tuples: %w", err)
	}
	return nil
}

// tupleInsert formats the tuple insert message for the given tuples.
func (br *Brain) tupleInsert(tuples []brain.Tuple) string {
	data := struct {
		Iter []struct{}
		// We don't actually have to pass the tuples to the template, we just
		// need the right number of elements.
		Tuples []struct{}
	}{
		Iter:   make([]struct{}, br.order),
		Tuples: make([]struct{}, len(tuples)),
	}
	var b strings.Builder
	if err := br.tpl.ExecuteTemplate(&b, "tuple.insert.sql", &data); err != nil {
		panic(err)
	}
	return b.String()
}

//go:embed message.insert.sql
var insertMessage string

//go:embed tuple.insert.sql
var insertTuple string
