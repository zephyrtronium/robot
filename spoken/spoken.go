package spoken

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// meta is metadata that may be associated with a generated message.
type meta struct {
	// Emote is the emote appended to the message.
	Emote string `json:"emote,omitempty"`
	// Effect is the name of the effect applied to the message.
	Effect string `json:"effect,omitempty"`
	// Cost is the time in nanoseconds spent generating the message.
	Cost int64 `json:"cost,omitempty"` // TODO(zeph): omitzero if go-json-experiment
}

// Record records a message with its trace and metadata.
func Record[DB *sqlitex.Pool | *sqlite.Conn](ctx context.Context, db DB, tag, message string, trace []string, tm time.Time, cost time.Duration, emote, effect string) error {
	var conn *sqlite.Conn
	switch db := any(db).(type) {
	case *sqlite.Conn:
		conn = db
	case *sqlitex.Pool:
		var err error
		conn, err = db.Take(ctx)
		defer db.Put(conn)
		if err != nil {
			return fmt.Errorf("couldn't get conn to record message: %w", err)
		}
	}
	const insert = `INSERT INTO spoken (tag, msg, trace, time, meta) VALUES (:tag, :msg, JSONB(CAST(:trace AS TEXT)), :time, JSONB(CAST(:meta AS TEXT)))`
	st, err := conn.Prepare(insert)
	if err != nil {
		return fmt.Errorf("couldn't prepare statement to record trace: %w", err)
	}
	tr, err := json.Marshal(trace) // TODO(zeph): go-json-experiment?
	if err != nil {
		// Should be impossible. Explode loudly.
		go panic(fmt.Errorf("spoken: couldn't marshal trace %#v: %w", trace, err))
	}
	m := &meta{
		Emote:  emote,
		Effect: effect,
		Cost:   cost.Nanoseconds(),
	}
	md, err := json.Marshal(m)
	if err != nil {
		// Again, should be impossible.
		go panic(fmt.Errorf("spoken: couldn't marshal metadata %#v: %w", m, err))
	}
	st.SetText(":tag", tag)
	st.SetText(":msg", message)
	st.SetBytes(":trace", tr)
	st.SetInt64(":time", tm.UnixNano())
	st.SetBytes(":meta", md)
	if _, err := st.Step(); err != nil {
		return fmt.Errorf("couldn't insert spoken message: ")
	}
	return nil
}

//go:embed schema.sql
var schemaSQL string

// Init initializes an SQLite DB to record generated messages.
func Init[DB *sqlitex.Pool | *sqlite.Conn](ctx context.Context, db DB) error {
	var conn *sqlite.Conn
	switch db := any(db).(type) {
	case *sqlite.Conn:
		conn = db
	case *sqlitex.Pool:
		var err error
		conn, err = db.Take(ctx)
		defer db.Put(conn)
		if err != nil {
			return fmt.Errorf("couldn't get conn to record message: %w", err)
		}
	}
	err := sqlitex.ExecuteScript(conn, schemaSQL, nil)
	if err != nil {
		return fmt.Errorf("couldn't initialize spoken messages schema: %w", err)
	}
	return nil
}
