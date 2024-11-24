package brain_test

import (
	"context"
	"slices"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

type testLearner struct {
	msg     []brain.Message
	learned []brain.Tuple
}

func (t *testLearner) Learn(ctx context.Context, tag string, msg *brain.Message, tuples []brain.Tuple) error {
	t.msg = append(t.msg, *msg)
	t.learned = append(t.learned, tuples...)
	return nil
}

func (t *testLearner) Forget(ctx context.Context, tag, id string) error {
	return nil
}

func (t *testLearner) Recall(ctx context.Context, tag string, page string, out []brain.Message) (n int, next string, err error) {
	var k int
	if page != "" {
		k, err = strconv.Atoi(page)
		if err != nil {
			return 0, "", err
		}
	}
	n = copy(out, t.msg[k:])
	k += n
	if k >= len(t.msg) {
		return n, "", nil
	}
	return n, strconv.Itoa(k), nil
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
			err := brain.Learn(context.Background(), &l, "", &brain.Message{Text: c.msg})
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(c.want, l.learned); diff != "" {
				t.Errorf("learned the wrong things from %q:\n%s", c.msg, diff)
			}
		})
	}
}

func TestRecall(t *testing.T) {
	cases := []struct {
		name  string
		learn []brain.Message
	}{
		{
			name:  "empty",
			learn: []brain.Message{},
		},
		{
			name: "one",
			learn: []brain.Message{
				{
					ID:        "1",
					To:        "#bocchi",
					Sender:    userhash.Hash{1},
					Text:      "bocchi the rock!",
					Timestamp: 1,
				},
			},
		},
		{
			name: "many",
			learn: slices.Repeat([]brain.Message{
				{
					ID:        "1",
					To:        "#bocchi",
					Sender:    userhash.Hash{1},
					Text:      "bocchi the rock!",
					Timestamp: 1,
				},
			}, 65), // this number should be larger than the buffer size Recall uses
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var l testLearner
			for _, m := range c.learn {
				err := brain.Learn(context.Background(), &l, "kessoku", &m)
				if err != nil {
					t.Fatalf("learn failed: %v", err)
				}
			}
			got := make([]brain.Message, 0, len(c.learn))
			for m, err := range brain.Recall(context.Background(), &l, "kessoku") {
				if err != nil {
					t.Error(err)
				}
				got = append(got, m)
			}
			if diff := cmp.Diff(got, c.learn); diff != "" {
				t.Errorf("wrong recollection (+got/-want):\n%s", diff)
			}
		})
	}
}
