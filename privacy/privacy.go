package privacy

import (
	"context"
	"errors"

	"gitlab.com/zephyrtronium/sq"
)

// ErrPrivate is an error returned by Check when the user is in the list.
var ErrPrivate = errors.New("user is private")

// DB is the minimal database interface used by DBList.
type DB interface {
	Exec(context.Context, string, ...any) (sq.Result, error)
	QueryRow(context.Context, string, ...any) *sq.Row
}

// List is a List backed by an SQL database.
type List struct {
	db DB
}

// Open opens an existing privacy list in an SQL database.
func Open(ctx context.Context, db DB) (*List, error) {
	// TODO(zeph): check that the db has the right table?
	return &List{db: db}, nil
}

// Init initializes a list in an SQL database.
func Init(ctx context.Context, db DB) error {
	_, err := db.Exec(ctx, `CREATE TABLE Privacy (user TEXT PRIMARY KEY)`)
	return err
}

// Add adds a user to the database.
func (l *List) Add(ctx context.Context, user string) error {
	_, err := l.db.Exec(ctx, `INSERT INTO Privacy (user) VALUES (?)`, user)
	return err
}

// Remove removes a user from the database.
func (l *List) Remove(ctx context.Context, user string) error {
	_, err := l.db.Exec(ctx, `DELETE FROM Privacy WHERE user=?`, user)
	return err
}

// Check checks whether a user is in the database.
func (l *List) Check(ctx context.Context, user string) error {
	var ok bool
	err := l.db.QueryRow(ctx, `SELECT ? IN (SELECT user FROM Privacy)`, user).Scan(&ok)
	if err != nil {
		return err
	}
	if ok {
		return ErrPrivate
	}
	return nil
}
