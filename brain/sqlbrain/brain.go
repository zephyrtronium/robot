package sqlbrain

import (
	"bytes"
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
	stmts statements
	order int
}

type statements struct {
	// selectTuple selects a tuple with a given tag and current state.
	selectTuple *sq.Stmt
	// newTuple selects a single starting term with a given tag.
	newTuple *sq.Stmt
	// deleteTuple is a sequence of statements to remove a single tuple with a
	// given tag. It is strings instead of prepared statements because the
	// sqlite3 driver actively resists my attempts to do horrible things.
	deleteTuple []string
}

// DB encapsulates database methods a Brain requires to allow use of a DB or a
// single Conn.
type DB interface {
	Exec(ctx context.Context, query string, args ...any) (sq.Result, error)
	Query(ctx context.Context, query string, args ...any) (*sq.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) *sq.Row
	Begin(ctx context.Context, opts *sq.TxOptions) (*sq.Tx, error)
	Prepare(ctx context.Context, query string) (*sq.Stmt, error)
	Close() error
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
	// tuple.insert.sql is special because it is executed independently for
	// every call instead of being executed once and prepared.
	template.Must(br.tpl.New("tuple.insert.sql").Parse(insertTuple))
	br.stmts.newTuple = br.initTpStmt(ctx, "tuple.new.sql", newTuple)
	br.stmts.selectTuple = br.initTpStmt(ctx, "tuple.select.sql", selectTuple)
	br.initDelete()

	return &br, nil
}

// Close closes the brain's database.
func (br *Brain) Close() error {
	return br.db.Close()
}

// initTpStmt initializes a SQL statement that requires ahead-of-time template
// initialization. Panics on any error.
func (br *Brain) initTpStmt(ctx context.Context, name, text string) *sq.Stmt {
	fib := make([]int, br.order-1)
	a, b := 1, 1
	for i := range fib {
		a, b = b, a+b
		fib[i] = b
	}
	if br.order == 1 {
		// Special case for the minimum order. In this case, the minimum score
		// must be 0, because the score of every match is 0, since there is
		// nothing additional in the prefix to contribute score.
		a = 0
	}

	data := struct {
		Iter      []struct{}
		Fibonacci []int
		NM1       int
		MinScore  int
	}{
		Iter:      make([]struct{}, br.order),
		Fibonacci: fib,
		NM1:       br.order - 1,
		MinScore:  a,
	}
	buf := make([]byte, 0, 2048)
	w := bytes.NewBuffer(buf)

	tp, err := br.tpl.New(name).Parse(text)
	if err != nil {
		panic(fmt.Errorf("couldn't parse template %s: %w", name, err))
	}
	if err := tp.Execute(w, &data); err != nil {
		panic(fmt.Errorf("couldn't execute template %s: %w", name, err))
	}
	s, err := br.db.Prepare(ctx, w.String())
	if err != nil {
		panic(fmt.Errorf("couldn't prepare statement from %s: %w", name, err))
	}
	return s
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

//go:embed templates/*.create.sql templates/*.pragma.sql
var createFiles embed.FS
var createTemplates = template.Must(template.ParseFS(createFiles, "templates/*.sql"))

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
	files, err := fs.ReadDir(createFiles, "templates")
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
		return fmt.Errorf("couldn't exec %s\n%w", query.String(), err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("couldn't commit: %w", err)
	}
	return nil
}
