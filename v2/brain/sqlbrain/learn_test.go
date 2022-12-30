package sqlbrain_test

import (
	"context"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/zephyrtronium/robot/v2/brain"
	"github.com/zephyrtronium/robot/v2/brain/sqlbrain"
	"github.com/zephyrtronium/robot/v2/brain/userhash"
)

func TestLearn(t *testing.T) {
	type row struct {
		ID   uuid.UUID
		User userhash.Hash
		Tag  string
		Ts   int64
		Pn   []string
		Suf  string
	}
	cases := []struct {
		name  string
		order int
		msg   brain.MessageMeta
		tups  []brain.Tuple
		want  []row
	}{
		{
			name:  "2x1",
			order: 2,
			msg: brain.MessageMeta{
				ID:   uuid.UUID([16]byte{0: 1}),
				User: userhash.Hash{1: 2},
				Tag:  "tag",
				Time: time.UnixMilli(3),
			},
			tups: []brain.Tuple{
				{Prefix: []string{"a", "b"}, Suffix: "c"},
			},
			want: []row{
				{
					ID:   uuid.UUID([16]byte{0: 1}),
					User: userhash.Hash{1: 2},
					Tag:  "tag",
					Ts:   3,
					Pn:   []string{"a", "b"},
					Suf:  "c",
				},
			},
		},
		{
			name:  "2x2",
			order: 2,
			msg: brain.MessageMeta{
				ID:   uuid.UUID([16]byte{1: 1}),
				User: userhash.Hash{2: 2},
				Tag:  "tag",
				Time: time.UnixMilli(4),
			},
			tups: []brain.Tuple{
				{Prefix: []string{"u", "v"}, Suffix: "w"},
				{Prefix: []string{"v", "w"}, Suffix: "x"},
			},
			want: []row{
				{
					ID:   uuid.UUID([16]byte{1: 1}),
					User: userhash.Hash{2: 2},
					Tag:  "tag",
					Ts:   4,
					Pn:   []string{"u", "v"},
					Suf:  "w",
				},
				{
					ID:   uuid.UUID([16]byte{1: 1}),
					User: userhash.Hash{2: 2},
					Tag:  "tag",
					Ts:   4,
					Pn:   []string{"v", "w"},
					Suf:  "x",
				},
			},
		},
		{
			name:  "4x1",
			order: 4,
			msg: brain.MessageMeta{
				ID:   uuid.UUID([16]byte{2: 1}),
				User: userhash.Hash{3: 2},
				Tag:  "tag",
				Time: time.UnixMilli(5),
			},
			tups: []brain.Tuple{
				{Prefix: []string{"a", "b", "c", "d"}, Suffix: "e"},
			},
			want: []row{
				{
					ID:   uuid.UUID([16]byte{2: 1}),
					User: userhash.Hash{3: 2},
					Tag:  "tag",
					Ts:   5,
					Pn:   []string{"a", "b", "c", "d"},
					Suf:  "e",
				},
			},
		},
		{
			name:  "4x2",
			order: 4,
			msg: brain.MessageMeta{
				ID:   uuid.UUID([16]byte{3: 1}),
				User: userhash.Hash{4: 2},
				Tag:  "tag",
				Time: time.UnixMilli(6),
			},
			tups: []brain.Tuple{
				{Prefix: []string{"u", "v", "w", "x"}, Suffix: "y"},
				{Prefix: []string{"v", "w", "x", "y"}, Suffix: "z"},
			},
			want: []row{
				{
					ID:   uuid.UUID([16]byte{3: 1}),
					User: userhash.Hash{4: 2},
					Tag:  "tag",
					Ts:   6,
					Pn:   []string{"u", "v", "w", "x"},
					Suf:  "y",
				},
				{
					ID:   uuid.UUID([16]byte{3: 1}),
					User: userhash.Hash{4: 2},
					Tag:  "tag",
					Ts:   6,
					Pn:   []string{"v", "w", "x", "y"},
					Suf:  "z",
				},
			},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := testDB(c.order)
			br, err := sqlbrain.Open(ctx, db)
			if err != nil {
				t.Fatalf("couldn't open brain: %v", err)
			}
			if err := br.Learn(ctx, &c.msg, c.tups); err != nil {
				t.Errorf("couldn't learn: %v", err)
			}
			q := `SELECT id, user, tag, time, {{range $i, $_ := $}}p{{$i}}, {{end}}suffix FROM Message, Tuple`
			var b strings.Builder
			template.Must(template.New("query").Parse(q)).Execute(&b, make([]struct{}, c.order))
			rows, err := db.Query(ctx, b.String())
			if err != nil {
				t.Fatalf("couldn't select: %v", err)
			}
			for i := 0; rows.Next(); i++ {
				got := row{Pn: make([]string, c.order)}
				dst := []any{&got.ID, &got.User, &got.Tag, &got.Ts}
				for i := range got.Pn {
					dst = append(dst, &got.Pn[i])
				}
				dst = append(dst, &got.Suf)
				if err := rows.Scan(dst...); err != nil {
					t.Errorf("couldn't scan: %v", err)
				}
				if i >= len(c.want) {
					t.Errorf("too many rows: got %+v", got)
					continue
				}
				if diff := cmp.Diff(c.want[i], got); diff != "" {
					t.Errorf("got wrong row #%d: %s", i, diff)
				}
			}
		})
	}
}
