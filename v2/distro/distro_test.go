package distro_test

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zephyrtronium/robot/v2/distro"
)

func TestDistString(t *testing.T) {
	type pick struct {
		v uint32
		w string
	}
	cases := []struct {
		name  string
		elems []distro.Case[string]
		pick  []pick
	}{
		{
			name:  "empty",
			elems: nil,
			pick: []pick{
				{0, ""},
				{1, ""},
				{0x7fffffff, ""},
				{0x80000000, ""},
				{0xffffffff, ""},
			},
		},
		{
			name: "single",
			elems: []distro.Case[string]{
				{E: "madoka", W: 1},
			},
			pick: []pick{
				{0, "madoka"},
				{1, "madoka"},
				{0x7fffffff, "madoka"},
				{0x80000000, "madoka"},
				{0xffffffff, "madoka"},
			},
		},
		{
			name: "single-weighted",
			elems: []distro.Case[string]{
				{E: "madoka", W: 100},
			},
			pick: []pick{
				{0, "madoka"},
				{1, "madoka"},
				{0x7fffffff, "madoka"},
				{0x80000000, "madoka"},
				{0xffffffff, "madoka"},
			},
		},
		{
			name: "double",
			elems: []distro.Case[string]{
				{E: "madoka", W: 1},
				{E: "homura", W: 1},
			},
			pick: []pick{
				{0, "madoka"},
				{1, "madoka"},
				{0x7fffffff, "madoka"},
				{0x80000000, "homura"},
				{0xffffffff, "homura"},
			},
		},
		{
			name: "double-weighted",
			elems: []distro.Case[string]{
				{E: "madoka", W: 100},
				{E: "homura", W: 100},
			},
			pick: []pick{
				{0, "madoka"},
				{1, "madoka"},
				{0x7fffffff, "madoka"},
				{0x80000000, "homura"},
				{0xffffffff, "homura"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := distro.New(c.elems)
			for _, p := range c.pick {
				r := d.Pick(p.v)
				if r != p.w {
					t.Errorf("wrong pick with variate %#x: want %v, got %v", p.v, p.w, r)
				}
			}
		})
	}
}

func TestFromMap(t *testing.T) {
	cases := []struct {
		name  string
		elems map[string]int
		want  []distro.Case[string]
	}{
		{
			name:  "empty",
			elems: nil,
			want:  nil,
		},
		{
			name: "single",
			elems: map[string]int{
				"madoka": 1,
			},
			want: []distro.Case[string]{
				{E: "madoka", W: 1},
			},
		},
		{
			name: "double",
			elems: map[string]int{
				"madoka": 1,
				"homura": 2,
			},
			want: []distro.Case[string]{
				{E: "homura", W: 2},
				{E: "madoka", W: 1},
			},
		},
		{
			name: "zero",
			elems: map[string]int{
				"madoka": 1,
				"homura": 0,
			},
			want: []distro.Case[string]{
				{E: "madoka", W: 1},
			},
		},
		{
			name: "negative",
			elems: map[string]int{
				"madoka": 1,
				"homura": -1,
			},
			want: []distro.Case[string]{
				{E: "madoka", W: 1},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := distro.FromMap(c.elems)
			sort.Slice(r, func(i, j int) bool { return r[i].E < r[j].E })
			if diff := cmp.Diff(r, c.want, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("wrong cases (-want/+got):\n%s", diff)
			}
		})
	}
}
