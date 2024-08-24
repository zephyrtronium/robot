package sqlbrain_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/braintest"
	"github.com/zephyrtronium/robot/brain/sqlbrain"
)

var dbCount atomic.Int64

func testDB(ctx context.Context) *sqlitex.Pool {
	k := dbCount.Add(1)
	pool, err := sqlitex.NewPool(fmt.Sprintf("file:%d.db?mode=memory&cache=shared", k), sqlitex.PoolOptions{Flags: sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenMemory | sqlite.OpenSharedCache | sqlite.OpenURI})
	if err != nil {
		panic(err)
	}
	conn, err := pool.Take(ctx)
	defer pool.Put(conn)
	if err != nil {
		panic(err)
	}
	return pool
}

var _ brain.Learner = (*sqlbrain.Brain)(nil)
var _ brain.Speaker = (*sqlbrain.Brain)(nil)

func TestIntegrated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	new := func(ctx context.Context) brain.Brain {
		db := testDB(ctx)
		br, err := sqlbrain.Open(ctx, db)
		if err != nil {
			panic(err)
		}
		return br
	}
	braintest.Test(ctx, t, new)
}
