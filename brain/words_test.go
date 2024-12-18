package brain

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/google/go-cmp/cmp"
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
			want: nil,
		},
		{
			name: "single",
			msg:  "bocchi",
			want: s("bocchi "),
		},
		{
			name: "append",
			msg:  "ryo",
			in:   s("bocchi"),
			want: s("bocchi", "ryo "),
		},
		{
			name: "split",
			msg:  "bocchi ryo nijika kita",
			want: s("bocchi ", "ryo ", "nijika ", "kita "),
		},
		{
			name: "punct",
			msg:  "'bocchi' 'ryo'",
			want: s("'", "bocchi", "' ", "'", "ryo", "' "),
		},
		{
			name: "colon",
			msg:  "bocchi: ryo",
			want: s("bocchi", ": ", "ryo "),
		},
		{
			name: "at",
			msg:  "@bocchi ryo",
			want: s("@bocchi ", "ryo "),
		},
		{
			name: "at-end",
			msg:  "bocchi@",
			want: s("bocchi", "@ "),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := tokens(c.in, c.msg)
			if diff := cmp.Diff(c.want, got); diff != "" {
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
		dst = tokens(dst[:0], msgs[rand.Uint32()%uint32(len(msgs))])
	}
}
