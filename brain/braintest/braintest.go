// Package braintest provides integration testing facilities for brains.
package braintest

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

// Interface is a merged learner and speaker for testing.
type Interface interface {
	brain.Learner
	brain.Speaker
}

// Test runs the integration test suite against brains produced by new.
//
// If a brain cannot be created without error, new should call t.Fatal.
func Test(ctx context.Context, t *testing.T, new func(context.Context) Interface) {
	t.Run("combinatoric", testCombinatoric(ctx, new(ctx)))
}

// testCombinatoric tests that chains can generate even with substantial
// overlap in learned material.
func testCombinatoric(ctx context.Context, br Interface) func(t *testing.T) {
	return func(t *testing.T) {
		msg := brain.MessageMeta{
			ID:   uuid.UUID{1},
			User: userhash.Hash{2},
			Tag:  "bocchi",
			Time: time.Unix(0, 0),
		}
		band := []string{"bocchi", "ryou", "nijika", "kita"}
		toks := make([]string, 6)
		for _, toks[0] = range band {
			for _, toks[1] = range band {
				for _, toks[2] = range band {
					for _, toks[3] = range band {
						for _, toks[4] = range band {
							for _, toks[5] = range band {
								toks := toks
								for len(toks) > 1 {
									err := brain.Learn(ctx, br, &msg, toks)
									if err != nil {
										t.Fatalf("couldn't learn init: %v", err)
									}
									toks = toks[1:]
								}
							}
						}
					}
				}
			}
		}
		allocs := testing.AllocsPerRun(100, func() {
			_, err := brain.Speak(ctx, br, "bocchi", "")
			if err != nil {
				t.Errorf("couldn't speak: %v", err)
			}
		})
		t.Logf("speaking cost %v allocs per run", allocs)
	}
}
