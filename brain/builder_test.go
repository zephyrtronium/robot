package brain_test

import (
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"

	"github.com/zephyrtronium/robot/brain"
)

func TestBuilder(t *testing.T) {
	cases := []struct {
		name  string
		terms [][2]string
		want  string
		trace []string
	}{
		{
			name:  "empty",
			terms: nil,
			want:  "",
			trace: nil,
		},
		{
			name: "single",
			terms: [][2]string{
				{"bocchi", "ryo"},
			},
			want: "ryo",
			trace: []string{
				"bocchi",
			},
		},
		{
			name: "multi",
			terms: [][2]string{
				{"bocchi", "ryo"},
				{"nijika", "kita"},
			},
			want: "ryokita",
			trace: []string{
				"bocchi",
				"nijika",
			},
		},
		{
			name: "order",
			terms: [][2]string{
				{"nijika", "ryo"},
				{"bocchi", "kita"},
			},
			want: "ryokita",
			trace: []string{
				"bocchi",
				"nijika",
			},
		},
		{
			name: "dedup",
			terms: [][2]string{
				{"bocchi", "ryo"},
				{"bocchi", "kita"},
			},
			want: "ryokita",
			trace: []string{
				"bocchi",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var b brain.Builder
			for _, t := range c.terms {
				b.Append(t[0], []byte(t[1]))
			}
			got := b.String()
			trace := b.Trace()
			if got != c.want {
				t.Errorf("wrong string: want %q, got %q", c.want, got)
			}
			if !slices.Equal(trace, c.trace) {
				t.Errorf("wrong trace: want %q, got %q", c.trace, trace)
			}
			b.Reset()
			got = b.String()
			trace = b.Trace()
			if got != "" {
				t.Errorf("string %q not empty after reset", got)
			}
			if len(trace) != 0 {
				t.Errorf("trace %q not empty after reset", trace)
			}
		})
	}
}

func BenchmarkBuilder(b *testing.B) {
	var ids [256]string
	var words [256][]byte
	for i := range ids {
		ids[i] = strconv.FormatUint(rand.Uint64()>>(i/16), 2)
		words[i] = []byte(ids[i])
	}
	var m brain.Builder
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		m.Reset()
		u := rand.Uint64()
		m.Append(ids[byte(u>>0)], words[byte(u>>8)])
		m.Append(ids[byte(u>>16)], words[byte(u>>24)])
		m.Append(ids[byte(u>>32)], words[byte(u>>40)])
		m.Append(ids[byte(u>>48)], words[byte(u>>56)])
	}
}
