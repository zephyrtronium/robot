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
	append []byte
}

func (t *testSpeaker) Speak(ctx context.Context, tag string, prompt []string, w []byte) ([]byte, error) {
	t.prompt = prompt
	return append(w, t.append...), nil
}

func TestSpeak(t *testing.T) {
	cases := []struct {
		name   string
		prompt string
		append []byte
		want   []string
		say    string
	}{
		{
			name:   "empty",
			prompt: "",
			append: nil,
			want:   nil,
			say:    "",
		},
		{
			name:   "empty-add",
			prompt: "",
			append: []byte("bocchi"),
			want:   nil,
			say:    "bocchi",
		},
		{
			name:   "prompted",
			prompt: "bocchi ryo nijika",
			append: nil,
			want:   []string{"nijika ", "ryo ", "bocchi "},
			say:    "bocchi ryo nijika",
		},
		{
			name:   "prompted-add",
			prompt: "bocchi ryo nijika",
			append: []byte("kita"),
			want:   []string{"nijika ", "ryo ", "bocchi "},
			say:    "bocchi ryo nijika kita",
		},
		{
			name:   "entropy",
			prompt: "BOCCHI RYO NIJIKA",
			append: []byte("KITA"),
			want:   []string{"nijika ", "ryo ", "bocchi "},
			say:    "BOCCHI RYO NIJIKA KITA",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := testSpeaker{append: c.append}
			r, err := brain.Speak(context.Background(), &s, "", c.prompt)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(c.want, s.prompt, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("wrong prompt from %q:\n%s", c.prompt, diff)
			}
			if diff := cmp.Diff(c.say, r); diff != "" {
				t.Errorf("wrong result from %q:\n%s", c.say, diff)
			}
		})
	}
}
