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
	t.Run("forgetful", testForgetful(ctx, new(ctx)))
	t.Run("combinatoric", testCombinatoric(ctx, new(ctx)))
}

// testForgetful tests that a brain forgets what it forgets.
func testForgetful(ctx context.Context, br Interface) func(t *testing.T) {
	return func(t *testing.T) {
		id := uuid.UUID{1}
		tag := "bocchi"
		msg := brain.MessageMeta{
			ID:   id,
			User: userhash.Hash{2},
			Tag:  tag,
			Time: time.Unix(0, 0),
		}
		text := "bocchi ryou nijika kita"
		toks := brain.Tokens(nil, text)
		if err := brain.Learn(ctx, br, &msg, toks); err != nil {
			t.Errorf("failed to learn: %v", err)
		}
		s, err := brain.Speak(ctx, br, tag, "")
		if err != nil {
			t.Errorf("failed to speak: %v", err)
		}
		if s != text {
			t.Errorf("surprise thought: %q", s)
		}
		if err := brain.Forget(ctx, br, tag, toks); err != nil {
			t.Errorf("failed to forget: %v", err)
		}
		// We don't really care about an error here, since the brain is empty.
		// All we care about is no thoughts.
		s, _ = brain.Speak(ctx, br, tag, "")
		if s != "" {
			t.Errorf("remembered that which must be forgotten: %q", s)
		}
	}
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
