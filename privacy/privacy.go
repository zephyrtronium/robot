package privacy

import (
	"context"
	"errors"
	"fmt"

	"zombiezen.com/go/sqlite/sqlitex"
)

// ErrPrivate is an error returned by Check when the user is in the list.
var ErrPrivate = errors.New("user is private")

// List is a List backed by an SQL database.
type List struct {
	db *sqlitex.Pool
}

// Open opens an existing privacy list in an SQL database.
func Open(ctx context.Context, db *sqlitex.Pool) (*List, error) {
	conn, err := db.Take(ctx)
	defer db.Put(conn)
	if err != nil {
		return nil, fmt.Errorf("couldn't get connection from pool: %w", err)
	}
	const schemaSQL = `CREATE TABLE IF NOT EXISTS privacy (user TEXT PRIMARY KEY) STRICT, WITHOUT ROWID`
	if err := sqlitex.ExecuteTransient(conn, schemaSQL, nil); err != nil {
		return nil, fmt.Errorf("couldn't run migration: %w", err)
	}
	return &List{db: db}, nil
}

// Add adds a user to the database.
func (l *List) Add(ctx context.Context, user string) error {
	conn, err := l.db.Take(ctx)
	defer l.db.Put(conn)
	if err != nil {
		return fmt.Errorf("couldn't get connection to add user to privacy list: %w", err)
	}
	opts := sqlitex.ExecOptions{Args: []any{user}}
	err = sqlitex.Execute(conn, `INSERT INTO privacy (user) VALUES (?)`, &opts)
	return err
}

// Remove removes a user from the database.
func (l *List) Remove(ctx context.Context, user string) error {
	conn, err := l.db.Take(ctx)
	defer l.db.Put(conn)
	if err != nil {
		return fmt.Errorf("couldn't get connection to remove user from privacy list: %w", err)
	}
	opts := sqlitex.ExecOptions{Args: []any{user}}
	err = sqlitex.Execute(conn, `DELETE FROM privacy WHERE user=?`, &opts)
	return err
}

// Check checks whether a user is in the database.
func (l *List) Check(ctx context.Context, user string) error {
	conn, err := l.db.Take(ctx)
	defer l.db.Put(conn)
	if err != nil {
		return fmt.Errorf("couldn't get connection to check user privacy: %w", err)
	}
	st, err := conn.Prepare(`SELECT ? IN (SELECT user FROM privacy)`)
	if err != nil {
		return fmt.Errorf("couldn't prepare statement to check user privacy: %w", err)
	}
	st.BindText(1, user)
	ok, err := sqlitex.ResultBool(st)
	if err != nil {
		return err
	}
	if ok {
		return ErrPrivate
	}
	return nil
}
