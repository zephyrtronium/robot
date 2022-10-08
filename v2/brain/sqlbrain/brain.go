package sqlbrain

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	_ "github.com/mattn/go-sqlite3" // driver
	"gitlab.com/zephyrtronium/sq"
)

type Brain struct {
	db    DB
	tpl   *template.Template
	order int
}

// DB encapsulates database methods a Brain requires to allow use of a DB or a
// single Conn.
type DB interface {
	Exec(ctx context.Context, query string, args ...any) (sq.Result, error)
	Query(ctx context.Context, query string, args ...any) (*sq.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) *sq.Row
	Begin(ctx context.Context, opts *sq.TxOptions) (*sq.Tx, error)
}

var _, _ DB = (*sq.DB)(nil), (*sq.Conn)(nil)

// Open returns a brain within the given database. The db must remain open for
// the lifetime of the brain.
func Open(ctx context.Context, db DB) (*Brain, error) {
	br := Brain{
		db:  db,
		tpl: template.New("base"),
	}
	err := db.QueryRow(ctx, `SELECT value FROM Config WHERE option='order'`).Scan(&br.order)
	if err != nil {
		return nil, fmt.Errorf("couldn't get order from database (not a brain?): %w", err)
	}
	// Parse templates.
	template.Must(br.tpl.New("tuple.insert.sql").Parse(insertTuple))

	return &br, nil
}

// Order returns the brain's configured Markov chain order.
func (br *Brain) Order() int {
	return br.order
}

// Tx opens a transaction directly with the brain's database. Passing nil for
// opts uses reasonable defaults. The returned transaction must be committed
// or rolled back once finished.
func (br *Brain) Tx(ctx context.Context, opts *sq.TxOptions) (*sq.Tx, error) {
	return br.db.Begin(ctx, opts)
}

//go:embed *.create.sql *.pragma.sql
var createFiles embed.FS
var createTemplates = template.Must(template.ParseFS(createFiles, "*.sql"))

// Create initializes a new brain with the given order within a database.
func Create(ctx context.Context, db DB, order int) error {
	// Create the query to generate the right schema with the given order.
	data := struct {
		N, NM1  int
		Version int
		Iter    []struct{}
	}{
		N: order, NM1: order - 1,
		Version: SchemaVersion,
		Iter:    make([]struct{}, order),
	}
	var query strings.Builder
	files, err := fs.ReadDir(createFiles, ".")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		err := createTemplates.ExecuteTemplate(&query, file.Name(), &data)
		if err != nil {
			// A problem here is a problem with the templates.
			panic(fmt.Errorf("couldn't interpret %s: %w", file.Name(), err))
		}
	}
	// Execute the query.
	tx, err := db.Begin(ctx, nil)
	if err != nil {
		return fmt.Errorf("couldn't open transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, query.String()); err != nil {
		return fmt.Errorf("couldn't exec: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("couldn't commit: %w", err)
	}
	return nil
}
