package prepend

import (
	"fmt"
	"slices"
	"testing"
)

func TestPrepender(t *testing.T) {
	cases := []struct {
		name string
		set  []int
		pre  [][]int
		drop int
		want []int
	}{
		{
			name: "empty",
			set:  nil,
			pre:  nil,
			drop: 0,
			want: nil,
		},
		{
			name: "set",
			set:  []int{1, 2},
			pre:  nil,
			drop: 0,
			want: []int{1, 2},
		},
		{
			name: "empty-pre",
			set:  nil,
			pre:  [][]int{{1}},
			drop: 0,
			want: []int{1},
		},
		{
			name: "pre",
			set:  []int{2},
			pre:  [][]int{{1}},
			drop: 0,
			want: []int{1, 2},
		},
		{
			name: "many-pre",
			set:  nil,
			pre:  [][]int{{1}, {2}, {3}, {4}, {5}, {6}, {7}, {8}, {9}, {10}, {11}, {12}, {13}, {14}, {15}, {16}},
			drop: 0,
			// prepending gives reverse order
			want: []int{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		},
		{
			name: "multi-pre",
			set:  nil,
			pre:  [][]int{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}},
			drop: 0,
			want: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			name: "empty-drop",
			set:  nil,
			pre:  nil,
			drop: 1,
			want: nil,
		},
		{
			name: "drop",
			set:  []int{1, 2},
			pre:  nil,
			drop: 1,
			want: []int{1},
		},
		{
			name: "drop-minus",
			set:  []int{1, 2},
			pre:  nil,
			drop: -1,
			want: []int{1, 2},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var p *List[int]
			invariants := func(step string) {
				if want, got := len(p.Slice()), p.Len(); want != got {
					t.Errorf("lengths disagree after %s: slice gives %d, len gives %d", step, want, got)
				}
			}
			invariants("nil decl")
			p = p.Set(c.set...)
			invariants("set")
			for _, x := range c.pre {
				p = p.Prepend(x...)
				invariants(fmt.Sprintf("prepend %d", x))
			}
			p = p.Drop(c.drop)
			invariants("drop")
			if !slices.Equal(p.Slice(), c.want) {
				t.Errorf("wrong final list:\nwant %v\ngot  %v", c.want, p.Slice())
			}
			p = p.Reset()
			invariants("reset")
			if len(p.Slice()) != 0 {
				t.Errorf("not empty after reset: %v", p.Slice())
			}
		})
	}
}
