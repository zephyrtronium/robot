package sqlbrain_test

import (
	"context"
	"testing"

	"github.com/zephyrtronium/robot/v2/brain"
	"github.com/zephyrtronium/robot/v2/brain/sqlbrain"
	"gitlab.com/zephyrtronium/sq"

	_ "github.com/mattn/go-sqlite3" // driver for tests
)

func testDB(order int) sqlbrain.DB {
	ctx := context.Background()
	db, err := sq.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	if err := db.Ping(ctx); err != nil {
		panic(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		panic(err)
	}
	if err := sqlbrain.Create(ctx, conn, order); err != nil {
		panic(err)
	}
	return conn
}

func TestOpen(t *testing.T) {
	ctx := context.Background()
	br, err := sqlbrain.Open(ctx, testDB(2))
	if err != nil {
		t.Error(err)
	}
	if got := br.Order(); got != 2 {
		t.Errorf("wrong order after opening brain: want 2, got %d", got)
	}
}

var _ brain.Learner = (*sqlbrain.Brain)(nil)
var _ brain.Speaker = (*sqlbrain.Brain)(nil)
