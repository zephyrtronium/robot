package sqlbrain_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"gitlab.com/zephyrtronium/sq"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/braintest"
	"github.com/zephyrtronium/robot/brain/sqlbrain"
	"github.com/zephyrtronium/robot/userhash"
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

func BenchmarkLearn(b *testing.B) {
	dir := filepath.ToSlash(b.TempDir())
	for _, order := range []int{2, 4, 6, 16} {
		b.Run(strconv.Itoa(order), func(b *testing.B) {
			new := func(ctx context.Context, b *testing.B) brain.Learner {
				dsn := fmt.Sprintf("file:%s/benchmark_learn_%d.db?_journal=WAL&_mutex=full", dir, order)
				db, err := sq.Open("sqlite3", dsn)
				if err != nil {
					b.Fatal(err)
				}
				// The benchmark function will run this multiple times to
				// estimate iteration count, so we need to drop tables and
				// views if they exist.
				stmts := []string{
					`DROP VIEW IF EXISTS MessageTuple`,
					`DROP TABLE IF EXISTS Tuple`,
					`DROP TABLE IF EXISTS Message`,
					`DROP TABLE IF EXISTS Config`,
				}
				for _, s := range stmts {
					_, err := db.Exec(context.Background(), s)
					if err != nil {
						b.Fatal(err)
					}
				}
				if err := sqlbrain.Create(context.Background(), db, order); err != nil {
					b.Fatal(err)
				}
				br, err := sqlbrain.Open(ctx, db)
				if err != nil {
					b.Fatalf("couldn't open brain: %v", err)
				}
				return br
			}
			braintest.BenchLearn(context.Background(), b, new, func(l brain.Learner) { l.(*sqlbrain.Brain).Close() })
		})
	}
}
