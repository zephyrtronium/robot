package sqlbrain

import (
	"context"
	_ "embed"
	"fmt"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// Forget forgets everything learned from a single given message.
// If nothing has been learned from the message, a message with that ID cannot
// be learned in the future.
func (br *Brain) Forget(ctx context.Context, tag, id string) (err error) {
	conn, err := br.db.Take(ctx)
	defer br.db.Put(conn)
	if err != nil {
		return fmt.Errorf("couldn't get connection to forget message %v: %w", id, err)
	}
	defer sqlitex.Transaction(conn)(&err)
	{
		// First forget the message, so that an attempt to learn it later will fail.
		const forget = `
			INSERT INTO messages (tag, id, deleted) VALUES (:tag, :id, 'CLEARMSG')
			ON CONFLICT DO UPDATE SET deleted = 'CLEARMSG'
		`
		st, err := conn.Prepare(forget)
		if err != nil {
			return fmt.Errorf("couldn't prepare delete for message %v: %w", id, err)
		}
		st.SetText(":tag", tag)
		st.SetText(":id", id)
		if err := allsteps(st); err != nil {
			return fmt.Errorf("couldn't delete message %v: %w", id, err)
		}
	}
	{
		// Now forget tuples.
		const forget = `UPDATE knowledge SET deleted = 'CLEARMSG' WHERE tag=:tag AND id=:id`
		st, err := conn.Prepare(forget)
		if err != nil {
			return fmt.Errorf("couldn't prepare delete for tuples of message %v: %w", id, err)
		}
		st.SetText(":tag", tag)
		st.SetText(":id", id)
		if err := allsteps(st); err != nil {
			return fmt.Errorf("couldn't delete tuples of message %v: %w", id, err)
		}
	}
	return nil
}

func allsteps(st *sqlite.Stmt) error {
	for {
		ok, err := st.Step()
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
}
