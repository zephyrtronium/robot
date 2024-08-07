package kvbrain_test

import (
	"context"
	"testing"

	"github.com/dgraph-io/badger/v4"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/braintest"
	"github.com/zephyrtronium/robot/brain/kvbrain"
)

func TestBrain(t *testing.T) {
	braintest.Test(context.Background(), t, func(ctx context.Context) brain.Brain {
		db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil))
		if err != nil {
			t.Fatal(err)
		}
		return kvbrain.New(db)
	})
}
