package sqlbrain

import (
	"context"
	_ "embed"
	"fmt"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// Brain is an implementation of knowledge using an SQLite database.
type Brain struct {
	db *sqlitex.Pool
}

// Open returns a brain within the given database.
// The db must remain open for the lifetime of the brain.
func Open(ctx context.Context, db *sqlitex.Pool) (*Brain, error) {
	// TODO(zeph): validate schema
	br := Brain{db}
	return &br, nil
}

//go:embed schema.sql
var schemaSQL string

// Create initializes a new brain in a database.
// For convenience, it accepts either a single connection or a pool.
func Create[DB *sqlite.Conn | *sqlitex.Pool](ctx context.Context, db DB) error {
	var conn *sqlite.Conn
	switch db := any(db).(type) {
	case *sqlite.Conn:
		conn = db
	case *sqlitex.Pool:
		var err error
		conn, err = db.Take(ctx)
		defer db.Put(conn)
		if err != nil {
			return fmt.Errorf("couldn't get connection from pool: %w", err)
		}
	}
	err := sqlitex.ExecuteScript(conn, schemaSQL, nil)
	if err != nil {
		return fmt.Errorf("couldn't initialize schema: %w", err)
	}
	return nil
}

// Close closes the underlying database.
func (br *Brain) Close() error {
	return br.db.Close()
}
