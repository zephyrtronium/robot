package brain_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zephyrtronium/robot/v2/brain"
)

type testSpeaker struct {
	order  int
	new    []string
	prompt []string
}

func (t *testSpeaker) Order() int {
	return t.order
}

func (t *testSpeaker) New(ctx context.Context, tag string) ([]string, error) {
	return t.new, nil
}

func (t *testSpeaker) Speak(ctx context.Context, tag string, prompt []string) ([]string, error) {
	// TODO(zeph): use ReduceEntropy
	t.prompt = prompt
	return prompt, nil
}

func TestSpeak(t *testing.T) {
	cases := []struct {
		name   string
		prompt string
		order  int
		new    []string
		want   []string
	}{
		{
			name:   "empty",
			prompt: "",
			order:  1,
			new:    nil,
			want:   nil,
		},
		{
			name:   "empty-new",
			prompt: "",
			order:  1,
			new:    []string{"anime"},
			want:   []string{"anime"},
		},
		{
			name:   "prompted-short",
			prompt: "madoka",
			order:  2,
			want:   []string{"", "madoka"},
		},
		{
			name:   "prompted-even",
			prompt: "madoka homura",
			order:  2,
			want:   []string{"madoka", "homura"},
		},
		{
			name:   "prompted-long",
			prompt: "madoka homura anime",
			order:  2,
			want:   []string{"homura", "anime"},
		},
		// TODO(zeph): cases testing entropy reduction
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := testSpeaker{order: c.order, new: c.new}
			r, err := brain.Speak(context.Background(), &s, "", c.prompt)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(c.want, s.prompt); diff != "" {
				t.Errorf("wrong prompt from %q:\n%s", c.prompt, diff)
			}
			// Check that each word the speaker gave to Speak appears in
			// sequence in the result.
			t.Logf("%q gave result %q", c.prompt, r)
			for _, w := range c.want {
				if w == "" {
					continue
				}
				k := strings.Index(r, w)
				if k < 0 {
					t.Errorf("%q doesn't appear in %q", w, r)
					continue
				}
				r = r[k:]
			}
		})
	}
}

func TestSpeakResult(t *testing.T) {
	cases := []struct {
		name   string
		prompt string
		order  int
		new    []string
		want   string
	}{
		{
			name:   "long-prompt",
			prompt: "madoka homura sayaka mami",
			order:  1,
			want:   "madoka homura sayaka mami",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := testSpeaker{order: c.order, new: c.new}
			r, err := brain.Speak(context.Background(), &s, "", c.prompt)
			if err != nil {
				t.Error(err)
			}
			if r != c.want {
				t.Errorf("wrong result: wanted %q, got %q", c.want, r)
			}
		})
	}
}
