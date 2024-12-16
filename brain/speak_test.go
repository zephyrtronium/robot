package brain_test

import (
	"context"
	"iter"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/message"
	"github.com/zephyrtronium/robot/userhash"
)

type testSpeaker struct {
	prompt []string
	id     string
	append []byte
}

func (t *testSpeaker) Think(ctx context.Context, tag string, prefix []string) iter.Seq[func(id *[]byte, suf *[]byte) error] {
	panic("unimplemented")
}

func (t *testSpeaker) Speak(ctx context.Context, tag string, prompt []string, w *brain.Builder) error {
	t.prompt = prompt
	w.Append(t.id, t.append)
	return nil
}

// Forget implements brain.Brain.
func (t *testSpeaker) Forget(ctx context.Context, tag string, id string) error {
	panic("unimplemented")
}

// Learn implements brain.Brain.
func (t *testSpeaker) Learn(ctx context.Context, tag string, msg *message.Received[userhash.Hash], tuples []brain.Tuple) error {
	panic("unimplemented")
}

// Recall implements brain.Brain.
func (t *testSpeaker) Recall(ctx context.Context, tag string, page string, out []message.Received[userhash.Hash]) (n int, next string, err error) {
	panic("unimplemented")
}

func TestSpeak(t *testing.T) {
	cases := []struct {
		name   string
		prompt string
		id     string
		append []byte
		want   []string
		trace  []string
		say    string
	}{
		{
			name:   "empty",
			prompt: "",
			id:     "kessoku",
			append: nil,
			want:   nil,
			trace:  []string{"kessoku"},
			say:    "",
		},
		{
			name:   "empty-add",
			prompt: "",
			id:     "kessoku",
			append: []byte("bocchi"),
			want:   nil,
			trace:  []string{"kessoku"},
			say:    "bocchi",
		},
		{
			name:   "prompted",
			prompt: "bocchi ryo nijika",
			id:     "kessoku",
			append: nil,
			want:   []string{"nijika ", "ryo ", "bocchi "},
			trace:  []string{"kessoku"},
			say:    "bocchi ryo nijika",
		},
		{
			name:   "prompted-add",
			prompt: "bocchi ryo nijika",
			id:     "kessoku",
			append: []byte("kita"),
			want:   []string{"nijika ", "ryo ", "bocchi "},
			trace:  []string{"kessoku"},
			say:    "bocchi ryo nijika kita",
		},
		{
			name:   "entropy",
			prompt: "BOCCHI RYO NIJIKA",
			id:     "kessoku",
			append: []byte("KITA"),
			want:   []string{"nijika ", "ryo ", "bocchi "},
			trace:  []string{"kessoku"},
			say:    "BOCCHI RYO NIJIKA KITA",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := testSpeaker{id: c.id, append: c.append}
			r, trace, err := brain.Speak(context.Background(), &s, "", c.prompt)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(c.want, s.prompt, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("wrong prompt from %q:\n%s", c.prompt, diff)
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
