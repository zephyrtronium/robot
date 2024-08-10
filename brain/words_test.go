package brain_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zephyrtronium/robot/brain"
)

func TestWords(t *testing.T) {
	s := func(x ...string) []string { return x }
	cases := []struct {
		name string
		msg  string
		in   []string
		want []string
	}{
		{
			name: "empty",
			msg:  "",
			in:   nil,
			want: nil,
		},
		{
			name: "single",
			msg:  "bocchi",
			in:   nil,
			want: s("bocchi"),
		},
		{
			name: "append",
			msg:  "ryo",
			in:   s("bocchi"),
			want: s("bocchi", "ryo"),
		},
		{
			name: "split",
			msg:  "bocchi ryo nijika kita",
			in:   nil,
			want: s("bocchi", "ryo", "nijika", "kita"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := brain.Tokens(c.in, c.msg)
			if diff := cmp.Diff(got, c.want); diff != "" {
				t.Errorf("wrong result (+got/-want):\n%s", diff)
			}
		})
	}
}

func BenchmarkWords(b *testing.B) {
	var msgs [256]string
	terms := []string{"bocchi", "ryo", "nijika", "kita"}
	for i := range msgs {
		x := i % len(terms)
		y := i / len(terms) % len(terms)
		z := i / len(terms) / len(terms) % len(terms)
		w := i / len(terms) / len(terms) / len(terms) % len(terms)
		msgs[i] = fmt.Sprintf(`%s "%s" the %s... %s`, terms[x], terms[y], terms[z], terms[w])
	}
	dst := make([]string, 0, 16)
	b.ResetTimer()
	for range b.N {
		dst = brain.Tokens(dst[:0], msgs[rand.Uint32()%uint32(len(msgs))])
	}
}
