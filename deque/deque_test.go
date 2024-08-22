package deque_test

import (
	"slices"
	"testing"

	"github.com/zephyrtronium/robot/deque"
)

func TestDeque(t *testing.T) {
	cases := []struct {
		name    string
		append  []int
		prepend []int
		want    []int
	}{
		{
			name:    "empty",
			append:  nil,
			prepend: nil,
			want:    nil,
		},
		{
			name:    "append",
			append:  []int{1, 2},
			prepend: nil,
			want:    []int{1, 2},
		},
		{
			name:    "prepend",
			append:  nil,
			prepend: []int{1, 2},
			want:    []int{1, 2},
		},
		{
			name:    "both",
			append:  []int{1, 2},
			prepend: []int{3, 4},
			want:    []int{3, 4, 1, 2},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var d deque.Deque[int]
			invariants := func() {
				if d.Len() != len(d.Slice()) {
					t.Errorf("lens disagree: d.Len gave %d, len(d.Slice) gave %d", d.Len(), len(d.Slice()))
				}
			}
			invariants()
			d = d.Append(c.append...)
			invariants()
			d = d.Prepend(c.prepend...)
			invariants()
			if !slices.Equal(d.Slice(), c.want) {
				t.Errorf("wrong result: want %v, got %v", c.want, d.Slice())
			}
		})
	}
}

func TestDropEndWhile(t *testing.T) {
	cases := []struct {
		name  string
		start []bool
		want  []bool
	}{
		{
			name:  "empty",
			start: nil,
			want:  nil,
		},
		{
			name:  "none",
			start: []bool{false, false},
			want:  []bool{false, false},
		},
		{
			name:  "one",
			start: []bool{false, true},
			want:  []bool{false},
		},
		{
			name:  "end",
			start: []bool{true, false},
			want:  []bool{true, false},
		},
		{
			name:  "all",
			start: []bool{true, true},
			want:  nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := deque.Deque[bool]{}.Append(c.start...)
			d = d.DropEndWhile(func(b bool) bool { return b })
			if !slices.Equal(d.Slice(), c.want) {
				t.Errorf("wrong result: want %v, got %v", c.want, d.Slice())
			}
		})
	}
}
