package sqlbrain

import (
	"context"
	"fmt"
	"time"

	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

// Learn records a set of tuples.
func (br *Brain) Learn(ctx context.Context, tag, id string, user userhash.Hash, t time.Time, tuples []brain.Tuple) (err error) {
	conn, err := br.db.Take(ctx)
	defer br.db.Put(conn)
	if err != nil {
		return fmt.Errorf("couldn't get connection to learn: %w", err)
	}
	defer sqlitex.Transaction(conn)(&err)

	st, err := conn.Prepare(`INSERT INTO knowledge(tag, id, prefix, suffix) VALUES (:tag, :id, :prefix, :suffix)`)
	if err != nil {
		return fmt.Errorf("couldn't prepare tuple insert: %w", err)
	}
	p := make([]byte, 0, 256)
	s := make([]byte, 0, 32)
	for _, tt := range tuples {
		p = append(prefix(p[:0], tt.Prefix), 0)
		s = append(s[:0], tt.Suffix...)
		st.SetText(":tag", tag)
		st.SetText(":id", id)
		st.SetBytes(":prefix", p)
		st.SetBytes(":suffix", s)
		_, err := st.Step()
		if err != nil {
			return fmt.Errorf("couldn't insert tuple: %w", err)
		}
		st.Reset()
	}

	sm, err := conn.Prepare(`INSERT INTO messages(tag, id, time, user) VALUES (:tag, :id, :time, :user)`)
	if err != nil {
		return fmt.Errorf("couldn't prepare message insert: %w", err)
	}
	sm.SetText(":tag", tag)
	sm.SetText(":id", id)
	sm.SetInt64(":time", t.UnixNano())
	sm.SetBytes(":user", user[:])
	_, err = sm.Step()
	if err != nil {
		return fmt.Errorf("couldn't insert message: %w", err)
	}

	return nil
}

func prefix(b []byte, tup []string) []byte {
	for _, w := range tup {
		b = append(b, w...)
		b = append(b, 0)
	}
	return b
}
