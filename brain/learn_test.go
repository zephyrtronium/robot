package brain_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

type testLearner struct {
	learned []brain.Tuple
}

func (t *testLearner) Learn(ctx context.Context, tag, id string, user userhash.Hash, tm time.Time, tuples []brain.Tuple) error {
	t.learned = append(t.learned, tuples...)
	return nil
}

func (t *testLearner) Forget(ctx context.Context, tag, id string) error {
	return nil
}

func TestLearn(t *testing.T) {
	s := func(x ...string) []string { return x }
	cases := []struct {
		name string
		msg  string
		want []brain.Tuple
	}{
		{
			name: "single",
			msg:  "word",
			want: []brain.Tuple{
				{Prefix: s("word "), Suffix: ""},
				{Prefix: nil, Suffix: "word "},
			},
		},
		{
			name: "many",
			msg:  "many words in this message",
			want: []brain.Tuple{
				{Prefix: s("message ", "this ", "in ", "words ", "many "), Suffix: ""},
				{Prefix: s("this ", "in ", "words ", "many "), Suffix: "message "},
				{Prefix: s("in ", "words ", "many "), Suffix: "this "},
				{Prefix: s("words ", "many "), Suffix: "in "},
				{Prefix: s("many "), Suffix: "words "},
				{Prefix: nil, Suffix: "many "},
			},
		},
		{
			name: "entropy",
			msg:  "A",
			want: []brain.Tuple{
				{Prefix: s("a "), Suffix: ""},
				{Prefix: nil, Suffix: "A "},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var l testLearner
			err := brain.Learn(context.Background(), &l, "", "", userhash.Hash{}, time.Unix(0, 0), c.msg)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(c.want, l.learned); diff != "" {
				t.Errorf("learned the wrong things from %q:\n%s", c.msg, diff)
			}
		})
	}
}
