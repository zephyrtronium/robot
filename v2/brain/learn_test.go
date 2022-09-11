package brain_test

import (
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
