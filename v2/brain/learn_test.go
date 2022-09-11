package brain_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zephyrtronium/robot/v2/brain"
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
	order   int
	learned []brain.Tuple
	err     error
}

func (t *testLearner) Order() int {
	return t.order
}

func (t *testLearner) Learn(ctx context.Context, meta *brain.MessageMeta, tuples []brain.Tuple) error {
	t.learned = append(t.learned, tuples...)
	return t.err
}

func TestLearn(t *testing.T) {
	s := func(x ...string) []string { return x }
	cases := []struct {
		name  string
		msg   []string
		order int
		want  []brain.Tuple
	}{
		{
			name:  "single-1",
			msg:   s("word"),
			order: 1,
			want: []brain.Tuple{
				{Prefix: s(""), Suffix: "word"},
				{Prefix: s("word"), Suffix: ""},
			},
		},
		{
			name:  "single-3",
			msg:   s("word"),
			order: 3,
			want: []brain.Tuple{
				{Prefix: s("", "", ""), Suffix: "word"},
				{Prefix: s("", "", "word"), Suffix: ""},
			},
		},
		{
			name:  "many-1",
			msg:   s("many", "words", "in", "this", "message"),
			order: 1,
			want: []brain.Tuple{
				{Prefix: s(""), Suffix: "many"},
				{Prefix: s("many"), Suffix: "words"},
				{Prefix: s("words"), Suffix: "in"},
				{Prefix: s("in"), Suffix: "this"},
				{Prefix: s("this"), Suffix: "message"},
				{Prefix: s("message"), Suffix: ""},
			},
		},
		{
			name:  "many-3",
			msg:   s("many", "words", "in", "this", "message"),
			order: 3,
			want: []brain.Tuple{
				{Prefix: s("", "", ""), Suffix: "many"},
				{Prefix: s("", "", "many"), Suffix: "words"},
				{Prefix: s("", "many", "words"), Suffix: "in"},
				{Prefix: s("many", "words", "in"), Suffix: "this"},
				{Prefix: s("words", "in", "this"), Suffix: "message"},
				{Prefix: s("in", "this", "message"), Suffix: ""},
			},
		},
		{
			name:  "entropy",
			msg:   s("A"),
			order: 1,
			want: []brain.Tuple{
				{Prefix: s(""), Suffix: "A"},
				{Prefix: s("a"), Suffix: ""},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := testLearner{order: c.order}
			err := brain.Learn(context.Background(), &l, nil, c.msg)
			if err != nil {
				t.Error(err)
			}
			// Check lengths of prefixes against the order we put down rather
			// than leaving it to cmp, because I have been known to typo a test
			// case or two when writing them at 5 AM.
			for _, p := range l.learned {
				if len(p.Prefix) != c.order {
					t.Errorf("wrong prefix size: want %d, got %d", c.order, len(p.Prefix))
				}
			}
			if diff := cmp.Diff(c.want, l.learned); diff != "" {
				t.Errorf("learned the wrong things from %q:\n%s", c.msg, diff)
			}
		})
	}
}

func TestMinimumOrder(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Error("no panic")
		}
	}()
	brain.Learn(context.Background(), new(testLearner), nil, []string{"word"})
}
