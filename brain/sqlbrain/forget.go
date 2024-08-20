package sqlbrain

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/zephyrtronium/robot/userhash"
)

// ForgetMessage forgets everything learned from a single given message.
// If nothing has been learned from the message, nothing happens.
func (br *Brain) ForgetMessage(ctx context.Context, tag, id string) (err error) {
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

// ForgetDuring forgets all messages learned in the given time span.
func (br *Brain) ForgetDuring(ctx context.Context, tag string, since time.Time, before time.Time) error {
	conn, err := br.db.Take(ctx)
	defer br.db.Put(conn)
	if err != nil {
		return fmt.Errorf("couldn't get connection to forget time span: %w", err)
	}
	defer sqlitex.Transaction(conn)(&err)
	// Forget messages by time and get their IDs.
	const forgetTime = `UPDATE messages SET deleted = 'TIME' WHERE tag=:tag AND time BETWEEN :since AND :before RETURNING id`
	sm, err := conn.Prepare(forgetTime)
	if err != nil {
		return fmt.Errorf("couldn't prepare delete for messages in time span: %w", err)
	}
	sm.SetText(":tag", tag)
	sm.SetInt64(":since", since.UnixNano())
	sm.SetInt64(":before", before.UnixNano())
	const forgetTuple = `UPDATE knowledge SET deleted = 'TIME' WHERE tag=:tag AND id=:id`
	st, err := conn.Prepare(forgetTuple)
	if err != nil {
		return fmt.Errorf("couldn't prepare delete for tuples in time span: %w", err)
	}
	st.SetText(":tag", tag)
	// Now forget tuples by the IDs.
	for {
		ok, err := sm.Step()
		if err != nil {
			return fmt.Errorf("couldn't step delete for messages in time span: %w", err)
		}
		if !ok {
			break
		}
		id := sm.GetText("id")
		st.SetText(":id", id)
		if err := allsteps(st); err != nil {
			return fmt.Errorf("couldn't step delete for tuples in time span: %w", err)
		}
		if err := st.Reset(); err != nil {
			return fmt.Errorf("couldn't reset delete for tuples in time span: %w", err)
		}
	}
	return nil
}

// ForgetUser forgets all messages associated with a userhash.
func (br *Brain) ForgetUser(ctx context.Context, user *userhash.Hash) error {
	conn, err := br.db.Take(ctx)
	defer br.db.Put(conn)
	if err != nil {
		return fmt.Errorf("couldn't get connection to forget from user: %w", err)
	}
	defer sqlitex.Transaction(conn)(&err)
	// Forget messages by user and get their IDs.
	const forgetUser = `UPDATE messages SET deleted = 'CLEARCHAT' WHERE user = :user RETURNING tag, id`
	sm, err := conn.Prepare(forgetUser)
	if err != nil {
		return fmt.Errorf("couldn't prepare delete for messages from user: %w", err)
	}
	sm.SetBytes(":user", user[:])
	const forgetTuple = `UPDATE knowledge SET deleted = 'CLEARCHAT' WHERE tag=:tag AND id=:id`
	st, err := conn.Prepare(forgetTuple)
	if err != nil {
		return fmt.Errorf("couldn't prepare delete for tuples from user: %w", err)
	}
	// Now forget by the IDs.
	for {
		ok, err := sm.Step()
		if err != nil {
			return fmt.Errorf("couldn't step delete for messages from user: %w", err)
		}
		if !ok {
			break
		}
		tag := sm.GetText("tag")
		id := sm.GetText("id")
		st.SetText(":tag", tag)
		st.SetText(":id", id)
		if err := allsteps(st); err != nil {
			return fmt.Errorf("couldn't step delete for tuples from user: %w", err)
		}
		if err := st.Reset(); err != nil {
			return fmt.Errorf("couldn't reset delete for tuples from user: %w", err)
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
