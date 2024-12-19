package brain_test

import (
	"context"
	"iter"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/message"
	"github.com/zephyrtronium/robot/userhash"
)

func hasPrefix[S ~[]E, E comparable](long, short S) bool {
	if len(short) > len(long) {
		return false
	}
	return slices.Equal(long[:len(short)], short)
}

type testThinker struct {
	pres [][]string
	id   string
	tups []brain.Tuple
}

func (t *testThinker) Think(ctx context.Context, tag string, prefix []string) iter.Seq[func(id *[]byte, suf *[]byte) error] {
	t.pres = append(t.pres, slices.Clone(prefix))
	return func(yield func(func(id *[]byte, suf *[]byte) error) bool) {
		var w string
		f := func(id, suf *[]byte) error {
			*id = []byte(t.id)
			*suf = []byte(w)
			return nil
		}
		for _, v := range t.tups {
			if !hasPrefix(v.Prefix, prefix) {
				continue
			}
			w = v.Suffix
			if !yield(f) {
				break
			}
		}
	}
}

func (t *testThinker) Speak(ctx context.Context, tag string, prompt []string, w *brain.Builder) error {
	panic("TODO: remove")
}

// Forget implements brain.Brain.
func (t *testThinker) Forget(ctx context.Context, tag string, id string) error {
	panic("unimplemented")
}

// Learn implements brain.Brain.
func (t *testThinker) Learn(ctx context.Context, tag string, msg *message.Received[userhash.Hash], tuples []brain.Tuple) error {
	panic("unimplemented")
}

// Recall implements brain.Brain.
func (t *testThinker) Recall(ctx context.Context, tag string, page string, out []message.Received[userhash.Hash]) (n int, next string, err error) {
	panic("unimplemented")
}

func TestThink(t *testing.T) {
	cases := []struct {
		name   string
		prompt string
		id     string
		tups   []brain.Tuple
		pres   [][]string
		trace  []string
		say    string
	}{
		{
			name:   "empty",
			prompt: "",
			id:     "kessoku",
			tups:   nil,
			pres:   [][]string{{}},
			trace:  nil,
			say:    "",
		},
		{
			name:   "empty-add",
			prompt: "",
			id:     "kessoku",
			tups: []brain.Tuple{
				{
					Prefix: nil,
					Suffix: "bocchi",
				},
			},
			pres:  [][]string{{}, {"bocchi"}},
			trace: []string{"kessoku"},
			say:   "bocchi",
		},
		{
			name:   "prompted-empty",
			prompt: "bocchi ryo nijika",
			id:     "kessoku",
			tups:   nil,
			pres:   [][]string{{"nijika ", "ryo ", "bocchi "}},
			trace:  nil,
			say:    "",
		},
		{
			name:   "prompted-add",
			prompt: "bocchi ryo nijika",
			id:     "kessoku",
			tups: []brain.Tuple{
				{
					Prefix: []string{"nijika "},
					Suffix: "kita",
				},
			},
			pres:  [][]string{{"nijika ", "ryo ", "bocchi "}},
			trace: []string{"kessoku"},
			say:   "bocchi ryo nijika kita",
		},
		{
			name:   "entropy",
			prompt: "BOCCHI RYO NIJIKA",
			id:     "kessoku",
			tups: []brain.Tuple{
				{
					Prefix: []string{"nijika "},
					Suffix: "KITA",
				},
			},
			pres:  [][]string{{"nijika ", "ryo ", "bocchi "}},
			trace: []string{"kessoku"},
			say:   "BOCCHI RYO NIJIKA KITA",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := testThinker{id: c.id, tups: c.tups}
			r, trace, err := brain.Think(context.Background(), &s, "", c.prompt)
			if err != nil {
				t.Error(err)
			}
			for _, want := range c.pres {
				if !slices.ContainsFunc(s.pres, func(got []string) bool { return slices.Equal(got, want) }) {
					t.Errorf("missing prompt %q in %q", want, s.pres)
				}
			}
			if diff := cmp.Diff(c.say, r); diff != "" {
				t.Errorf("wrong result from %q:\n%s", c.say, diff)
			}
			if diff := cmp.Diff(c.trace, trace); diff != "" {
				t.Errorf("wrong trace:\n%s", diff)
			}
		})
	}
}
