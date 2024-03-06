package channel

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"
	"gitlab.com/zephyrtronium/sq"
)

// MemeDetector is literally a meme detector.
type MemeDetector struct {
	// db is a database of messages received and memes sent in a channel.
	db *sq.Conn
	// need is the number of messages needed to trigger memery.
	need int
	// within is the duration to hold messages.
	within time.Duration
}

var (
	memdb     *sq.DB
	memdbOnce sync.Once
)

func loadMemdb() {
	var err error
	memdb, err = sq.Open("sqlite3", ":memory:")
	if err != nil {
		panic(fmt.Errorf("couldn't open memory db: %w", err))
	}
	if err := memdb.Ping(context.Background()); err != nil {
		panic(fmt.Errorf("couldn't ping memory db: %w", err))
	}
}

// NewMemeDetector creates.
func NewMemeDetector(need int, within time.Duration) *MemeDetector {
	ctx := context.Background()
	memdbOnce.Do(loadMemdb)
	conn, err := memdb.Conn(ctx)
	if err != nil {
		panic(fmt.Errorf("couldn't open single conn: %w", err))
	}
	if _, err := conn.Exec(ctx, createCopypasta); err != nil {
		panic(fmt.Errorf("couldn't set up meme tables: %w", err))
	}
	return &MemeDetector{
		db:     conn,
		need:   need,
		within: within,
	}
}

func (m *MemeDetector) DB() *sq.Conn {
	return m.db
}

// Check determines whether a message is a meme. If it is not, the returned
// error is NotCopypasta. Times passed to Check should be monotonic, as
// messages outside the detector's threshold are removed.
func (m *MemeDetector) Check(t time.Time, from, msg string) error {
	ctx := context.Background()
	tm := t.UnixMilli()
	// Remove old messages.
	_, err := m.db.Exec(ctx, `DELETE FROM Message WHERE time < ?`, tm-m.within.Milliseconds())
	if err != nil {
		return fmt.Errorf("couldn't remove old messages from meme detector: %w", err)
	}
	// Discard old memes.
	_, err = m.db.Exec(ctx, `DELETE FROM Meme WHERE time < ?`, tm-15*time.Minute.Milliseconds())
	if err != nil {
		return fmt.Errorf("couldn't remove old memes: %w", err)
	}
	// Insert the new one.
	_, err = m.db.Exec(ctx, `INSERT INTO Message (time, user, msg) VALUES (?, ?, ?)`, tm, from, msg)
	if err != nil {
		return fmt.Errorf("couldn't insert new message into meme detector: %w", err)
	}
	// Get the meme metric: number of distinct users who sent this message in
	// the time window.
	var n int
	err = m.db.QueryRow(ctx, `SELECT COUNT(DISTINCT user) FROM Message WHERE msg = ?`, msg).Scan(&n)
	if err != nil {
		return fmt.Errorf("couldn't get memery count: %w", err)
	}
	if n < m.need {
		return ErrNotCopypasta
	}
	// Genuine meme. But is it fresh?
	_, err = m.db.Exec(ctx, `INSERT INTO Meme (time, msg) VALUES (?, ?)`, tm, msg)
	if err != nil {
		// Since we expect to react to (i.e. log) non-copypasta errors that
		// aren't ErrNotCopypasta, it's more helpful to return it when it's a
		// real reason not to be copypasta.
		if err, ok := err.(sqlite3.Error); ok {
			if err.Code == sqlite3.ErrConstraint && err.ExtendedCode == sqlite3.ErrConstraintUnique {
				return ErrNotCopypasta
			}
		}
		return fmt.Errorf("couldn't register fresh meme: %w %#v", err, err)
	}
	return nil
}

// ErrNotCopypasta is a sentinel error returned by MemeDetector.Check when a
// message is not copypasta.
var ErrNotCopypasta = errors.New("not copypasta")

//go:embed copypasta.sql
var createCopypasta string
