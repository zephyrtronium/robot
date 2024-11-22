// Package braintest provides integration testing facilities for brains.
package braintest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

// Test runs the integration test suite against brains produced by new.
//
// If a brain cannot be created without error, new should call t.Fatal.
func Test(ctx context.Context, t *testing.T, new func(context.Context) brain.Brain) {
	t.Run("speak", testSpeak(ctx, new(ctx)))
	t.Run("forgetMessage", testForget(ctx, new(ctx)))
	t.Run("combinatoric", testCombinatoric(ctx, new(ctx)))
}

var messages = [...]struct {
	ID   string
	User userhash.Hash
	Tag  string
	Time time.Time
	Text string
}{
	{
		ID:   "1",
		User: userhash.Hash{2},
		Tag:  "kessoku",
		Time: time.Unix(0, 0),
		Text: "member bocchi",
	},
	{
		ID:   "2",
		User: userhash.Hash{2},
		Tag:  "kessoku",
		Time: time.Unix(1, 0),
		Text: "member ryou",
	},
	{
		ID:   "3",
		User: userhash.Hash{3},
		Tag:  "kessoku",
		Time: time.Unix(2, 0),
		Text: "member nijika",
	},
	{
		ID:   "4",
		User: userhash.Hash{3},
		Tag:  "kessoku",
		Time: time.Unix(3, 0),
		Text: "member kita",
	},
	{
		ID:   "5",
		User: userhash.Hash{2},
		Tag:  "sickhack",
		Time: time.Unix(0, 0),
		Text: "member bocchi",
	},
	{
		ID:   "6",
		User: userhash.Hash{2},
		Tag:  "sickhack",
		Time: time.Unix(1, 0),
		Text: "member ryou",
	},
	{
		ID:   "7",
		User: userhash.Hash{3},
		Tag:  "sickhack",
		Time: time.Unix(2, 0),
		Text: "member nijika",
	},
	{
		ID:   "8",
		User: userhash.Hash{3},
		Tag:  "sickhack",
		Time: time.Unix(3, 0),
		Text: "member kita",
	},
	{
		ID:   "9",
		User: userhash.Hash{4},
		Tag:  "sickhack",
		Time: time.Unix(43, 0),
		Text: "manager seika",
	},
}

func learn(ctx context.Context, t *testing.T, br brain.Learner) {
	t.Helper()
	for _, m := range messages {
		msg := brain.Message{ID: m.ID, Sender: m.User, Timestamp: m.Time.UnixMilli(), Text: m.Text}
		if err := brain.Learn(ctx, br, m.Tag, &msg); err != nil {
			t.Fatalf("couldn't learn message %v: %v", m.ID, err)
		}
	}
}

func speak(ctx context.Context, t *testing.T, br brain.Speaker, tag, prompt string, iters int) map[string]struct{} {
	t.Helper()
	got := make(map[string]struct{}, 20)
	for range iters {
		s, trace, err := brain.Speak(ctx, br, tag, prompt)
		if err != nil {
			t.Errorf("couldn't speak: %v", err)
		}
		got[strings.Join(trace, " ")+"#"+s] = struct{}{}
	}
	return got
}

// testSpeak tests that a brain can speak what it has learned.
func testSpeak(ctx context.Context, br brain.Brain) func(t *testing.T) {
	return func(t *testing.T) {
		learn(ctx, t, br)
		got := speak(ctx, t, br, "kessoku", "", 2048)
		want := map[string]struct{}{
			"1#member bocchi":   {},
			"1 2#member bocchi": {},
			"1 3#member bocchi": {},
			"1 4#member bocchi": {},
			"1 2#member ryou":   {},
			"2#member ryou":     {},
			"2 3#member ryou":   {},
			"2 4#member ryou":   {},
			"1 3#member nijika": {},
			"2 3#member nijika": {},
			"3#member nijika":   {},
			"3 4#member nijika": {},
			"1 4#member kita":   {},
			"2 4#member kita":   {},
			"3 4#member kita":   {},
			"4#member kita":     {},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong spoken messages for kessoku (+got/-want):\n%s", diff)
		}
		got = speak(ctx, t, br, "sickhack", "", 2048)
		want = map[string]struct{}{
			"5#member bocchi":   {},
			"5 6#member bocchi": {},
			"5 7#member bocchi": {},
			"5 8#member bocchi": {},
			"5 6#member ryou":   {},
			"6#member ryou":     {},
			"6 7#member ryou":   {},
			"6 8#member ryou":   {},
			"5 7#member nijika": {},
			"6 7#member nijika": {},
			"7#member nijika":   {},
			"7 8#member nijika": {},
			"5 8#member kita":   {},
			"6 8#member kita":   {},
			"7 8#member kita":   {},
			"8#member kita":     {},
			"9#manager seika":   {},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong spoken messages for sickhack (+got/-want):\n%s", diff)
		}
		got = speak(ctx, t, br, "sickhack", "manager", 32)
		want = map[string]struct{}{
			"9#manager seika": {},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong prompted messages for sickhack (+got/-want):\n%s", diff)
		}
	}
}

// testForget tests that a brain can forget messages by ID.
func testForget(ctx context.Context, br brain.Brain) func(t *testing.T) {
	return func(t *testing.T) {
		learn(ctx, t, br)
		if err := br.Forget(ctx, "kessoku", messages[0].ID); err != nil {
			t.Errorf("failed to forget first message: %v", err)
		}
		got := speak(ctx, t, br, "kessoku", "", 2048)
		want := map[string]struct{}{
			// The current brains should delete the "member" with ID 1, but we
			// don't strictly require it.
			// This should change anyway once we stop deleting by tuples.
			"2#member ryou":     {},
			"2 3#member ryou":   {},
			"2 4#member ryou":   {},
			"2 3#member nijika": {},
			"3#member nijika":   {},
			"3 4#member nijika": {},
			"2 4#member kita":   {},
			"3 4#member kita":   {},
			"4#member kita":     {},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong messages after forgetting (+got/-want):\n%s", diff)
		}
		got = speak(ctx, t, br, "sickhack", "", 2048)
		want = map[string]struct{}{
			"5#member bocchi":   {},
			"5 6#member bocchi": {},
			"5 7#member bocchi": {},
			"5 8#member bocchi": {},
			"5 6#member ryou":   {},
			"6#member ryou":     {},
			"6 7#member ryou":   {},
			"6 8#member ryou":   {},
			"5 7#member nijika": {},
			"6 7#member nijika": {},
			"7#member nijika":   {},
			"7 8#member nijika": {},
			"5 8#member kita":   {},
			"6 8#member kita":   {},
			"7 8#member kita":   {},
			"8#member kita":     {},
			"9#manager seika":   {},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong messages in other tag after forgetting (+got/-want):\n%s", diff)
		}
	}
}

// testCombinatoric tests that chains can generate even with substantial
// overlap in learned material.
func testCombinatoric(ctx context.Context, br brain.Brain) func(t *testing.T) {
	return func(t *testing.T) {
		u := userhash.Hash{2}
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
									id := randid()
									msg := brain.Message{ID: id, Sender: u, Text: strings.Join(toks, " ")}
									err := brain.Learn(ctx, br, "bocchi", &msg)
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
		allocs := testing.AllocsPerRun(10, func() {
			_, _, err := brain.Speak(ctx, br, "bocchi", "")
			if err != nil {
				t.Errorf("couldn't speak: %v", err)
			}
		})
		t.Logf("speaking cost %v allocs per run", allocs)
	}
}
