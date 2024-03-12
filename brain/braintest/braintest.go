// Package braintest provides integration testing facilities for brains.
package braintest

import (
	"context"
	"maps"
	"strings"
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
	t.Run("speak", testSpeak(ctx, new(ctx)))
	t.Run("forget", testForget(ctx, new(ctx)))
	t.Run("forgetMessage", testForgetMessage(ctx, new(ctx)))
	t.Run("forgetDuring", testForgetDuring(ctx, new(ctx)))
	t.Run("combinatoric", testCombinatoric(ctx, new(ctx)))
}

var messages = [...]struct {
	brain.MessageMeta
	Tokens []string
}{
	{
		MessageMeta: brain.MessageMeta{
			ID:   uuid.UUID{1},
			User: userhash.Hash{2},
			Tag:  "kessoku",
			Time: time.Unix(0, 0),
		},
		Tokens: []string{"member", "bocchi"},
	},
	{
		MessageMeta: brain.MessageMeta{
			ID:   uuid.UUID{2},
			User: userhash.Hash{2},
			Tag:  "kessoku",
			Time: time.Unix(1, 0),
		},
		Tokens: []string{"member", "ryou"},
	},
	{
		MessageMeta: brain.MessageMeta{
			ID:   uuid.UUID{3},
			User: userhash.Hash{3},
			Tag:  "kessoku",
			Time: time.Unix(2, 0),
		},
		Tokens: []string{"member", "nijika"},
	},
	{
		MessageMeta: brain.MessageMeta{
			ID:   uuid.UUID{4},
			User: userhash.Hash{3},
			Tag:  "kessoku",
			Time: time.Unix(3, 0),
		},
		Tokens: []string{"member", "kita"},
	},
	{
		MessageMeta: brain.MessageMeta{
			ID:   uuid.UUID{5},
			User: userhash.Hash{2},
			Tag:  "sickhack",
			Time: time.Unix(0, 0),
		},
		Tokens: []string{"member", "bocchi"},
	},
	{
		MessageMeta: brain.MessageMeta{
			ID:   uuid.UUID{6},
			User: userhash.Hash{2},
			Tag:  "sickhack",
			Time: time.Unix(1, 0),
		},
		Tokens: []string{"member", "ryou"},
	},
	{
		MessageMeta: brain.MessageMeta{
			ID:   uuid.UUID{7},
			User: userhash.Hash{3},
			Tag:  "sickhack",
			Time: time.Unix(2, 0),
		},
		Tokens: []string{"member", "nijika"},
	},
	{
		MessageMeta: brain.MessageMeta{
			ID:   uuid.UUID{8},
			User: userhash.Hash{3},
			Tag:  "sickhack",
			Time: time.Unix(3, 0),
		},
		Tokens: []string{"member", "kita"},
	},
	{
		MessageMeta: brain.MessageMeta{
			ID:   uuid.UUID{9},
			User: userhash.Hash{4},
			Tag:  "sickhack",
			Time: time.Unix(43, 0),
		},
		Tokens: []string{"manager", "seika"},
	},
}

func learn(ctx context.Context, t *testing.T, br brain.Learner) {
	t.Helper()
	for _, m := range messages {
		if err := brain.Learn(ctx, br, &m.MessageMeta, m.Tokens); err != nil {
			t.Fatalf("couldn't learn message %v: %v", m.ID, err)
		}
	}
}

func speak(ctx context.Context, t *testing.T, br brain.Speaker, tag, prompt string, iters int) map[string]bool {
	t.Helper()
	got := make(map[string]bool, 5)
	for range iters {
		s, err := brain.Speak(ctx, br, tag, prompt)
		if err != nil {
			t.Errorf("couldn't speak: %v", err)
		}
		got[s] = true
	}
	return got
}

// testSpeak tests that a brain can speak what it has learned.
func testSpeak(ctx context.Context, br Interface) func(t *testing.T) {
	return func(t *testing.T) {
		learn(ctx, t, br)
		got := speak(ctx, t, br, "kessoku", "", 256)
		want := map[string]bool{
			"member bocchi": true,
			"member ryou":   true,
			"member nijika": true,
			"member kita":   true,
		}
		if !maps.Equal(got, want) {
			t.Errorf("wrong spoken messages for kessoku: want %v, got %v", want, got)
		}
		got = speak(ctx, t, br, "sickhack", "", 256)
		want = map[string]bool{
			"member bocchi": true,
			"member ryou":   true,
			"member nijika": true,
			"member kita":   true,
			"manager seika": true,
		}
		if !maps.Equal(got, want) {
			t.Errorf("wrong spoken messages for sickhack: want %v, got %v", want, got)
		}
		got = speak(ctx, t, br, "sickhack", "manager", 32)
		want = map[string]bool{
			"manager seika": true,
		}
		if !maps.Equal(got, want) {
			t.Errorf("wrong prompted messages for sickhack: want %v, got %v", want, got)
		}
	}
}

// testForget tests that a brain forgets what it forgets.
func testForget(ctx context.Context, br Interface) func(t *testing.T) {
	return func(t *testing.T) {
		learn(ctx, t, br)
		if err := brain.Forget(ctx, br, "kessoku", messages[0].Tokens); err != nil {
			t.Errorf("couldn't forget: %v", err)
		}
		for range 100 {
			s, err := brain.Speak(ctx, br, "kessoku", "")
			if err != nil {
				t.Errorf("couldn't speak: %v", err)
			}
			if strings.Contains(s, "bocchi") {
				t.Errorf("remembered that which must be forgotten: %q", s)
			}
		}
		for {
			s, err := brain.Speak(ctx, br, "sickhack", "")
			if err != nil {
				t.Errorf("couldn't speak: %v", err)
			}
			if strings.Contains(s, "bocchi") {
				break
			}
			// The failure condition is that this loop is infinite.
		}
	}
}

// testForgetMessage tests that a brain can forget messages by ID.
func testForgetMessage(ctx context.Context, br Interface) func(t *testing.T) {
	return func(t *testing.T) {
		learn(ctx, t, br)
		if err := br.ForgetMessage(ctx, "kessoku", messages[0].ID); err != nil {
			t.Errorf("failed to forget first message: %v", err)
		}
		got := speak(ctx, t, br, "kessoku", "", 256)
		want := map[string]bool{
			"member ryou":   true,
			"member nijika": true,
			"member kita":   true,
		}
		if !maps.Equal(got, want) {
			t.Errorf("wrong messages after forgetting: want %v, got %v", want, got)
		}
		got = speak(ctx, t, br, "sickhack", "", 256)
		want = map[string]bool{
			"member bocchi": true,
			"member ryou":   true,
			"member nijika": true,
			"member kita":   true,
			"manager seika": true,
		}
		if !maps.Equal(got, want) {
			t.Errorf("wrong messages in other tag after forgetting: want %v, got %v", want, got)
		}
	}
}

// testForgetDuring tests that a brain can forget messages in a time range.
func testForgetDuring(ctx context.Context, br Interface) func(t *testing.T) {
	return func(t *testing.T) {
		learn(ctx, t, br)
		if err := br.ForgetDuring(ctx, "kessoku", time.Unix(1, 0).Add(-time.Millisecond), time.Unix(2, 0).Add(time.Millisecond)); err != nil {
			t.Errorf("failed to forget: %v", err)
		}
		got := speak(ctx, t, br, "kessoku", "", 256)
		want := map[string]bool{
			"member bocchi": true,
			"member kita":   true,
		}
		if !maps.Equal(got, want) {
			t.Errorf("wrong messages after forgetting: want %v, got %v", want, got)
		}
		got = speak(ctx, t, br, "sickhack", "", 256)
		want = map[string]bool{
			"member bocchi": true,
			"member ryou":   true,
			"member nijika": true,
			"member kita":   true,
			"manager seika": true,
		}
		if !maps.Equal(got, want) {
			t.Errorf("wrong spoken messages for sickhack: want %v, got %v", want, got)
		}
	}
}

// TODO(zeph): testForgetUser

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
