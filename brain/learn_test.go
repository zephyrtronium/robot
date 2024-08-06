package brain_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

func TestTokens(t *testing.T) {
	// Just to make test cases a bit easier to write.
	s := func(x ...string) []string { return x }
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"single", "single", s("single")},
		{"many", "many words in this message", s("many", "words", "in", "this", "message")},
		{"a", "a word", s("a word")},
		{"an", "an word", s("an word")},
		{"the", "the word", s("the word")},
		{"aend", "word a", s("word", "a")},
		{"anend", "word an", s("word", "an")},
		{"theend", "word the", s("word", "the")},
		{"aaa", "a a a", s("a", "a", "a")},
		{"ananan", "an an an", s("an an", "an")},
		{"thethethe", "the the the", s("the the", "the")},
		{"meme", "a x y", s("a", "x", "y")},
		{"spaces", "x    y", s("x", "y")},
		{"tabs", "x\ty", s("x", "y")},
		{"unicode", "x\u2002y", s("x", "y")},
		{"spaceend", "x y ", s("x", "y")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dst := make([]string, len(c.want))
			p := &dst[0]
			got := brain.Tokens(dst[:0], c.in)
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("wrong tokens from %q:\n%s", c.in, diff)
			}
			if p != &dst[0] {
				t.Error("first element pointer changed")
			}
		})
	}
}

type testLearner struct {
	learned []brain.Tuple
	forgot  []brain.Tuple
	err     error
}

func (t *testLearner) Learn(ctx context.Context, tag string, user userhash.Hash, id uuid.UUID, tm time.Time, tuples []brain.Tuple) error {
	t.learned = append(t.learned, tuples...)
	return t.err
}

func (t *testLearner) Forget(ctx context.Context, tag string, tuples []brain.Tuple) error {
	t.forgot = tuples
	return nil
}

func (t *testLearner) ForgetMessage(ctx context.Context, tag string, msg uuid.UUID) error {
	return nil
}

func (t *testLearner) ForgetDuring(ctx context.Context, tag string, since, before time.Time) error {
	return nil
}

func (t *testLearner) ForgetUser(ctx context.Context, user *userhash.Hash) error {
	return nil
}

func TestLearn(t *testing.T) {
	s := func(x ...string) []string { return x }
	cases := []struct {
		name string
		msg  []string
		want []brain.Tuple
	}{
		{
			name: "single",
			msg:  s("word"),
			want: []brain.Tuple{
				{Prefix: s("word"), Suffix: ""},
				{Prefix: nil, Suffix: "word"},
			},
		},
		{
			name: "many",
			msg:  s("many", "words", "in", "this", "message"),
			want: []brain.Tuple{
				{Prefix: s("message", "this", "in", "words", "many"), Suffix: ""},
				{Prefix: s("this", "in", "words", "many"), Suffix: "message"},
				{Prefix: s("in", "words", "many"), Suffix: "this"},
				{Prefix: s("words", "many"), Suffix: "in"},
				{Prefix: s("many"), Suffix: "words"},
				{Prefix: nil, Suffix: "many"},
			},
		},
		{
			name: "entropy",
			msg:  s("A"),
			want: []brain.Tuple{
				{Prefix: s("a"), Suffix: ""},
				{Prefix: nil, Suffix: "A"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var l testLearner
			err := brain.Learn(context.Background(), &l, "", userhash.Hash{}, uuid.UUID{}, time.Unix(0, 0), c.msg)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(c.want, l.learned); diff != "" {
				t.Errorf("learned the wrong things from %q:\n%s", c.msg, diff)
			}
		})
	}
}

func TestForget(t *testing.T) {
	s := func(x ...string) []string { return x }
	cases := []struct {
		name string
		msg  []string
		want []brain.Tuple
	}{
		{
			name: "single",
			msg:  s("word"),
			want: []brain.Tuple{
				{Prefix: s("word"), Suffix: ""},
				{Prefix: nil, Suffix: "word"},
			},
		},
		{
			name: "many-1",
			msg:  s("many", "words", "in", "this", "message"),
			want: []brain.Tuple{
				{Prefix: s("message", "this", "in", "words", "many"), Suffix: ""},
				{Prefix: s("this", "in", "words", "many"), Suffix: "message"},
				{Prefix: s("in", "words", "many"), Suffix: "this"},
				{Prefix: s("words", "many"), Suffix: "in"},
				{Prefix: s("many"), Suffix: "words"},
				{Prefix: nil, Suffix: "many"},
			},
		},
		{
			name: "entropy",
			msg:  s("A"),
			want: []brain.Tuple{
				{Prefix: s("a"), Suffix: ""},
				{Prefix: nil, Suffix: "A"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var l testLearner
			err := brain.Forget(context.Background(), &l, "", c.msg)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(c.want, l.forgot); diff != "" {
				t.Errorf("forgot the wrong things from %q:\n%s", c.msg, diff)
			}
		})
	}
}
