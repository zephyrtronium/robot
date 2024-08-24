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
	conn, err := db.Take(ctx)
	defer db.Put(conn)
	if err != nil {
		return nil, fmt.Errorf("couldn't get connection from pool: %w", err)
	}
	if err := sqlitex.ExecuteScript(conn, schemaSQL, nil); err != nil {
		return nil, fmt.Errorf("couldn't run migration: %w", err)
	}
	br := Brain{db}
	return &br, nil
}

//go:embed schema.sql
var schemaSQL string

// Close closes the underlying database.
func (br *Brain) Close() error {
	return br.db.Close()
}

// RecommendedPrep is an [sqlitex.ConnPrepareFunc] that sets options recommended
// for a brain.
func RecommendedPrep(conn *sqlite.Conn) error {
	// Performance pragmas.
	// These need to be run per connection.
	pragmas := []string{
		"PRAGMA journal_mode = WAL", // Should be set by the connection, but make really sure.
		"PRAGMA synchronous = NORMAL",
	}
	for _, p := range pragmas {
		s, _, err := conn.PrepareTransient(p)
		if err != nil {
			// If this function just returns an error, then the pool
			// will continue to invoke it for every connection.
			// Explode violently instead.
			panic(fmt.Errorf("couldn't set %s: %w", p, err))
		}
		if err := allsteps(s); err != nil {
			// This one is probably ok to retry, though.
			return fmt.Errorf("couldn't run %s: %w", p, err)
		}
		if err := s.Finalize(); err != nil {
			panic(fmt.Errorf("couldn't finalize statement for %s: %w", p, err))
		}
	}
	return nil
}
