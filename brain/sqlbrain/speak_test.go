package sqlbrain_test

import (
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/braintest"
	"github.com/zephyrtronium/robot/brain/sqlbrain"
)

func TestSpeak(t *testing.T) {
	cases := []struct {
		name   string
		know   []know
		tag    string
		prompt []string
		want   []string
	}{
		{
			name:   "empty",
			know:   nil,
			tag:    "kessoku",
			prompt: nil,
			// We should only ever get nil from the brain,
			// but that converts to the empty string.
			want: []string{""},
		},
		{
			name: "empty-tagged",
			know: []know{
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "bocchi ",
				},
				{
					tag:    "kessoku",
					prefix: "bocchi \x00",
					suffix: "",
				},
			},
			tag:    "sickhack",
			prompt: nil,
			// We should only ever get nil from the brain,
			// but that converts to the empty string.
			want: []string{""},
		},
		{
			name: "empty-prompted",
			know: []know{
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "bocchi ",
				},
				{
					tag:    "kessoku",
					prefix: "bocchi \x00",
					suffix: "",
				},
			},
			tag:    "kessoku",
			prompt: []string{"kikuri "},
			// We should only ever get nil from the brain,
			// but that converts to the empty string.
			want: []string{""},
		},
		{
			name: "single",
			know: []know{
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "bocchi ",
				},
				{
					tag:    "kessoku",
					prefix: "bocchi \x00",
					suffix: "",
				},
			},
			tag:    "kessoku",
			prompt: nil,
			want:   []string{"bocchi "},
		},
		{
			name: "several",
			know: []know{
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "bocchi ",
				},
				{
					tag:    "kessoku",
					prefix: "bocchi \x00",
					suffix: "ryo ",
				},
				{
					tag:    "kessoku",
					prefix: "ryo \x00bocchi \x00",
					suffix: "nijika ",
				},
				{
					tag:    "kessoku",
					prefix: "nijika \x00ryo \x00bocchi \x00",
					suffix: "kita ",
				},
				{
					tag:    "kessoku",
					prefix: "kita \x00nijika \x00ryo \x00bocchi \x00",
					suffix: "",
				},
			},
			tag:    "kessoku",
			prompt: nil,
			want:   []string{"bocchi ryo nijika kita "},
		},
		{
			name: "multi",
			know: []know{
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "kessoku",
					prefix: "member \x00",
					suffix: "bocchi ",
				},
				{
					tag:    "kessoku",
					prefix: "bocchi \x00member \x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "kessoku",
					prefix: "member \x00",
					suffix: "ryo ",
				},
				{
					tag:    "kessoku",
					prefix: "ryo \x00member \x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "kessoku",
					prefix: "member \x00",
					suffix: "nijika ",
				},
				{
					tag:    "kessoku",
					prefix: "nijika \x00member \x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "kessoku",
					prefix: "member \x00",
					suffix: "kita ",
				},
				{
					tag:    "kessoku",
					prefix: "kita \x00member \x00",
					suffix: "",
				},
			},
			tag:    "kessoku",
			prompt: nil,
			want:   []string{"member bocchi ", "member ryo ", "member nijika ", "member kita "},
		},
		{
			name: "multi-tagged",
			know: []know{
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "kessoku",
					prefix: "member \x00",
					suffix: "bocchi ",
				},
				{
					tag:    "kessoku",
					prefix: "bocchi \x00member \x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "kessoku",
					prefix: "member \x00",
					suffix: "ryo ",
				},
				{
					tag:    "kessoku",
					prefix: "ryo \x00member \x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "kessoku",
					prefix: "member \x00",
					suffix: "nijika ",
				},
				{
					tag:    "kessoku",
					prefix: "nijika \x00member \x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "kessoku",
					prefix: "member \x00",
					suffix: "kita ",
				},
				{
					tag:    "kessoku",
					prefix: "kita \x00member \x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "sickhack",
					prefix: "member \x00",
					suffix: "kikuri ",
				},
				{
					tag:    "sickhack",
					prefix: "kikuri \x00member \x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "sickhack",
					prefix: "member \x00",
					suffix: "eliza ",
				},
				{
					tag:    "sickhack",
					prefix: "eliza \x00member \x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "sickhack",
					prefix: "member \x00",
					suffix: "shima ",
				},
				{
					tag:    "sickhack",
					prefix: "shima \x00member \x00",
					suffix: "",
				},
			},
			tag:    "sickhack",
			prompt: nil,
			want:   []string{"member kikuri ", "member eliza ", "member shima "},
		},
		{
			name: "forgort",
			know: []know{
				{
					tag:     "kessoku",
					prefix:  "",
					suffix:  "member",
					deleted: ref("FORGET"),
				},
				{
					tag:    "kessoku",
					prefix: "member \x00",
					suffix: "bocchi ",
				},
				{
					tag:    "kessoku",
					prefix: "bocchi \x00member \x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "kessoku",
					prefix: "member \x00",
					suffix: "ryo ",
				},
				{
					tag:    "kessoku",
					prefix: "ryo \x00member \x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:    "kessoku",
					prefix: "member \x00",
					suffix: "nijika ",
				},
				{
					tag:    "kessoku",
					prefix: "nijika \x00member \x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					prefix: "",
					suffix: "member ",
				},
				{
					tag:     "kessoku",
					prefix:  "member \x00",
					suffix:  "kita ",
					deleted: ref("FORGET"),
				},
				{
					tag:     "kessoku",
					prefix:  "kita \x00member \x00",
					suffix:  "",
					deleted: ref("FORGET"),
				},
			},
			tag:    "kessoku",
			prompt: nil,
			want:   []string{"member bocchi ", "member ryo ", "member nijika "},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := testDB(ctx)
			br, err := sqlbrain.Open(ctx, db)
			if err != nil {
				t.Fatalf("couldn't open brain: %v", err)
			}
			conn, err := db.Take(ctx)
			defer db.Put(conn)
			if err != nil {
				t.Fatalf("couldn't get conn: %v", err)
			}
			insert(t, conn, c.know, nil)
			slices.Sort(c.want)
			got := make([]string, 0, len(c.want))
			var w brain.Builder
			for range 10000 {
				w.Reset()
				err := br.Speak(ctx, c.tag, c.prompt, &w)
				if err != nil {
					t.Errorf("couldn't speak: %v", err)
				}
				s := w.String()
				k, ok := slices.BinarySearch(got, s)
				if !ok {
					got = slices.Insert(got, k, s)
					if len(got) == len(c.want) {
						break
					}
				}
			}
			if !slices.Equal(c.want, got) {
				t.Errorf("wrong results:\nwant %q\ngot  %q", c.want, got)
			}
		})
	}
}

func insert(t *testing.T, conn *sqlite.Conn, know []know, msgs []msg) {
	t.Helper()
	for _, v := range know {
		opts := sqlitex.ExecOptions{
			Named: map[string]any{
				":tag":    v.tag,
				":id":     v.id[:],
				":prefix": []byte(v.prefix),
				":suffix": []byte(v.suffix),
			},
		}
		var err error
		if v.deleted != nil {
			opts.Named[":deleted"] = *v.deleted
			err = sqlitex.Execute(conn, `INSERT INTO knowledge(tag, id, prefix, suffix, deleted) VALUES (:tag, :id, :prefix, :suffix, :deleted)`, &opts)
		} else {
			err = sqlitex.Execute(conn, `INSERT INTO knowledge(tag, id, prefix, suffix) VALUES (:tag, :id, :prefix, :suffix)`, &opts)
		}
		if err != nil {
			t.Errorf("couldn't learn knowledge %v %q %q: %v", v.id, v.prefix, v.suffix, err)
		}
	}
	for _, v := range msgs {
		opts := sqlitex.ExecOptions{
			Named: map[string]any{
				":tag":  v.tag,
				":id":   v.id[:],
				":time": v.time,
				":user": v.user[:],
			},
		}
		var err error
		if v.deleted != nil {
			opts.Named[":deleted"] = *v.deleted
			err = sqlitex.Execute(conn, `INSERT INTO message(tag, id, time, user, deleted) VALUES (:tag, :id, time, :user, :deleted)`, &opts)
		} else {
			err = sqlitex.Execute(conn, `INSERT INTO message(tag, id, time, user) VALUES (:tag, :id, time, :user)`, &opts)
		}
		if err != nil {
			t.Errorf("couldn't learn message %v: %v", v.id, err)
		}
	}
}

func BenchmarkSpeak(b *testing.B) {
	var dbs atomic.Uint64
	new := func(ctx context.Context, b *testing.B) brain.Brain {
		k := dbs.Add(1)
		db, err := sqlitex.NewPool(fmt.Sprintf("file:%s/bench-%d.sql", b.TempDir(), k), sqlitex.PoolOptions{PrepareConn: sqlbrain.RecommendedPrep})
		if err != nil {
			b.Fatal(err)
		}
		if err := sqlbrain.Create(ctx, db); err != nil {
			b.Fatal(err)
		}
		br, err := sqlbrain.Open(ctx, db)
		if err != nil {
			b.Fatal(err)
		}
		return br
	}
	cleanup := func(l brain.Brain) {
		br := l.(*sqlbrain.Brain)
		br.Close()
	}
	braintest.BenchSpeak(context.Background(), b, new, cleanup)
}
