package sqlbrain_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/sqlbrain"
	"gitlab.com/zephyrtronium/sq"

	_ "github.com/mattn/go-sqlite3" // driver for tests
)

var testdbCounter atomic.Uint64

func testDB(order int) sqlbrain.DB {
	ctx := context.Background()
	k := testdbCounter.Add(1)
	db, err := sq.Open("sqlite3", fmt.Sprintf("file:%d.db?cache=shared&mode=memory", k))
	if err != nil {
		panic(err)
	}
	if err := db.Ping(ctx); err != nil {
		panic(err)
	}
	if err := sqlbrain.Create(ctx, db, order); err != nil {
		panic(err)
	}
	return db
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
