package spoken_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zephyrtronium/robot/spoken"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

var dbCount atomic.Int64

func testDB(ctx context.Context) *sqlitex.Pool {
	k := dbCount.Add(1)
	pool, err := sqlitex.NewPool(fmt.Sprintf("file:test-record-%d.db?mode=memory&cache=shared", k), sqlitex.PoolOptions{Flags: sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenMemory | sqlite.OpenSharedCache | sqlite.OpenURI})
	if err != nil {
		panic(err)
	}
	if err := spoken.Init(ctx, pool); err != nil {
		panic(err)
	}
	return pool
}

func TestRecord(t *testing.T) {
	ctx := context.Background()
	db := testDB(ctx)
	conn, err := db.Take(ctx)
	defer db.Put(conn)
	if err != nil {
		t.Fatalf("couldn't get conn: %v", err)
	}
	err = spoken.Record(ctx, db, "kessoku", "bocchi ryo", []string{"1", "2"}, time.Unix(1, 0), time.Second, "xD", "o")
	if err != nil {
		t.Errorf("couldn't record: %v", err)
	}

	opts := sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			tag := stmt.ColumnText(0)
			msg := stmt.ColumnText(1)
			trace := stmt.ColumnText(2)
			tm := stmt.ColumnInt64(3)
			meta := stmt.ColumnText(4)

			if tag != "kessoku" {
				t.Errorf("wrong tag recorded: want %q, got %q", "kessoku", tag)
			}
			if msg != "bocchi ryo" {
				t.Errorf("wrong message recorded: want %q, got %q", "bocchi ryo", msg)
			}
			var tr []string
			if err := json.Unmarshal([]byte(trace), &tr); err != nil {
				t.Errorf("couldn't unmarshal trace from %q: %v", trace, err)
			}
			if !slices.Equal(tr, []string{"1", "2"}) {
				t.Errorf("wrong trace recorded: want %q, got %q from %q", []string{"1", "2"}, tr, trace)
			}
			if got, want := time.Unix(0, tm), time.Unix(1, 0); got != want {
				t.Errorf("wrong time: want %v, got %v", want, got)
			}
			var md map[string]any
			if err := json.Unmarshal([]byte(meta), &md); err != nil {
				t.Errorf("couldn't unmarshal metadata from %q: %v", meta, md)
			}
			want := map[string]any{
				"emote":  "xD",
				"effect": "o",
				"cost":   float64(time.Second.Nanoseconds()),
			}
			if !maps.Equal(md, want) {
				t.Errorf("wrong metadata recorded: want %v, got %v from %q", want, md, meta)
			}
			return nil
		},
	}
	err = sqlitex.ExecuteTransient(conn, `SELECT tag, msg, JSON(trace), time, JSON(meta) FROM spoken`, &opts)
	if err != nil {
		t.Errorf("failed to scan: %v", err)
	}
}
