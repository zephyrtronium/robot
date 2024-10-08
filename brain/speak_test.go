package brain_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/zephyrtronium/robot/brain"
)

type testSpeaker struct {
	prompt []string
	id     string
	append []byte
}

func (t *testSpeaker) Speak(ctx context.Context, tag string, prompt []string, w *brain.Builder) error {
	t.prompt = prompt
	w.Append(t.id, t.append)
	return nil
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
