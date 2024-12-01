package sqlbrain

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"

	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

// Learn records a set of tuples.
func (br *Brain) Learn(ctx context.Context, tag string, msg *brain.Message, tuples []brain.Tuple) (err error) {
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
		st.SetText(":id", msg.ID)
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
	sm.SetText(":id", msg.ID)
	sm.SetInt64(":time", msg.Timestamp*1e6) // scale from milliseconds to nanoseconds for historical reasons
	sm.SetBytes(":user", msg.Sender[:])
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

// Recall fills out with messages read from the brain.
func (br *Brain) Recall(ctx context.Context, tag string, page string, out []brain.Message) (n int, next string, err error) {
	t, s, err := pageparams(page)
	if err != nil {
		return 0, "", err
	}

	conn, err := br.db.Take(ctx)
	defer br.db.Put(conn)
	if err != nil {
		return 0, "", fmt.Errorf("couldn't get connection to recall: %w", err)
	}

	st, err := conn.Prepare(recallQuery)
	if err != nil {
		return 0, "", fmt.Errorf("couldn't prepare recollection: %w", err)
	}
	st.SetText(":tag", tag)
	st.SetInt64(":startTime", t)
	st.SetText(":startID", s)
	st.SetInt64(":n", int64(len(out)))
	for i := range out {
		ok, err := st.Step()
		if err != nil {
			return 0, page, fmt.Errorf("couldn't step recollection: %w", err)
		}
		if !ok {
			out = out[:i]
			break
		}
		var u userhash.Hash
		s = st.ColumnText(0)
		t = st.ColumnInt64(1)
		st.ColumnBytes(2, u[:])
		out[i] = brain.Message{
			ID:        s,
			Timestamp: t / 1e6, // convert ns to ms
			Sender:    u,
			Text:      st.ColumnText(3),
		}
	}

	if err = st.Reset(); err != nil {
		// Return the error along with our normal results below.
		err = fmt.Errorf("resetting recollection statement failed: %w", err)
	}
	if len(out) == 0 {
		// No results. Recollection has ended.
		// This also happens if we were given zero elements to fill,
		// but that's the caller's problem.
		return 0, "", err
	}
	return len(out), topage(t, s), err
}

func pageparams(page string) (int64, string, error) {
	if page == "" {
		return 0, "", nil
	}
	r, err := strconv.QuotedPrefix(page)
	if err != nil {
		return 0, "", fmt.Errorf("bad page %q", page)
	}
	l := page[len(r):]
	t, err := strconv.ParseInt(l, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("bad page %q", page)
	}
	id, err := strconv.Unquote(r)
	if err != nil {
		return 0, "", fmt.Errorf("bad page %q", page)
	}
	return t, id, nil
}

func topage(t int64, id string) string {
	b := make([]byte, 0, 64)
	b = strconv.AppendQuoteToASCII(b, id)
	b = strconv.AppendInt(b, t, 10)
	return string(b)
}

//go:embed recall.sql
var recallQuery string
