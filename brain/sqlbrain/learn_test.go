package sqlbrain_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/braintest"
	"github.com/zephyrtronium/robot/brain/sqlbrain"
	"github.com/zephyrtronium/robot/userhash"
)

type learn struct {
	tag  string
	user userhash.Hash
	id   string
	t    int64
	tups []brain.Tuple
}

type know struct {
	tag     string
	id      string
	prefix  string
	suffix  string
	deleted *string
}

type msg struct {
	tag     string
	id      string
	time    int64
	user    userhash.Hash
	deleted *string
}

func ref[T any](x T) *T { return &x }

func contents(t *testing.T, conn *sqlite.Conn, know []know, msgs []msg) {
	t.Helper()
	for _, want := range know {
		opts := sqlitex.ExecOptions{
			Named: map[string]any{
				":tag":    want.tag,
				":id":     want.id,
				":prefix": []byte(want.prefix),
				":suffix": []byte(want.suffix),
			},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				n := stmt.ColumnInt64(0)
				if n != 1 {
					t.Errorf("wrong number of rows for tag=%q id=%v prefix=%q suffix=%q: want 1, got %d", want.tag, want.id, want.prefix, want.suffix, n)
				}
				return nil
			},
		}
		if want.deleted == nil {
			err := sqlitex.Execute(conn, `SELECT COUNT(*) FROM knowledge WHERE tag=:tag AND id=:id AND prefix=:prefix AND suffix=:suffix AND deleted IS NULL`, &opts)
			if err != nil {
				t.Errorf("couldn't check for tag=%q id=%v prefix=%q suffix=%q: %v", want.tag, want.id, want.prefix, want.suffix, err)
			}
		} else {
			opts.Named[":deleted"] = *want.deleted
			err := sqlitex.Execute(conn, `SELECT COUNT(*) FROM knowledge WHERE tag=:tag AND id=:id AND prefix=:prefix AND suffix=:suffix AND deleted=:deleted`, &opts)
			if err != nil {
				t.Errorf("couldn't check for tag=%q id=%v prefix=%q suffix=%q: %v", want.tag, want.id, want.prefix, want.suffix, err)
			}
		}
	}
	for _, want := range msgs {
		opts := sqlitex.ExecOptions{
			Named: map[string]any{
				":tag":  want.tag,
				":id":   want.id,
				":time": want.time,
				":user": want.user[:],
			},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				n := stmt.ColumnInt64(0)
				if n != 1 {
					t.Errorf("wrong number of rows for tag=%q id=%v time=%d user=%02x: want 1, got %d", want.tag, want.id, want.time, want.user, n)
				}
				return nil
			},
		}
		if want.deleted == nil {
			err := sqlitex.Execute(conn, `SELECT COUNT(*) FROM messages WHERE tag=:tag AND id=:id AND time=:time AND user=:user AND deleted IS NULL`, &opts)
			if err != nil {
				t.Errorf("couldn't check for tag=%q id=%v time=%d user=%02x deleted=null: %v", want.tag, want.id, want.time, want.user, err)
			}
		} else {
			opts.Named[":deleted"] = *want.deleted
			err := sqlitex.Execute(conn, `SELECT COUNT(*) FROM messages WHERE tag=:tag AND id=:id AND time=:time AND user=:user AND deleted=:deleted`, &opts)
			if err != nil {
				t.Errorf("couldn't check for tag=%q id=%v time=%d user=%02x deleted=%s: %v", want.tag, want.id, want.time, want.user, *want.deleted, err)
			}
		}
	}
}

func TestLearn(t *testing.T) {
	cases := []struct {
		name  string
		learn []learn
		know  []know
		msgs  []msg
	}{
		{
			name:  "empty",
			learn: nil,
			know:  nil,
			msgs:  nil,
		},
		{
			name: "terms",
			learn: []learn{
				{
					tag:  "kessoku",
					user: userhash.Hash{1},
					id:   "2",
					t:    3,
					tups: []brain.Tuple{
						{Prefix: strings.Fields("kita nijika ryo bocchi"), Suffix: ""},
						{Prefix: strings.Fields("nijika ryo bocchi"), Suffix: "kita"},
						{Prefix: strings.Fields("ryo bocchi"), Suffix: "nijika"},
						{Prefix: strings.Fields("bocchi"), Suffix: "ryo"},
						{Prefix: nil, Suffix: "bocchi"},
					},
				},
			},
			know: []know{
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "kita\x00nijika\x00ryo\x00bocchi\x00\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "nijika\x00ryo\x00bocchi\x00\x00",
					suffix: "kita",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "ryo\x00bocchi\x00\x00",
					suffix: "nijika",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "bocchi\x00\x00",
					suffix: "ryo",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "\x00",
					suffix: "bocchi",
				},
			},
			msgs: []msg{
				{
					tag:  "kessoku",
					id:   "2",
					time: 3e6,
					user: userhash.Hash{1},
				},
			},
		},
		{
			name: "unicode",
			learn: []learn{
				{
					tag:  "結束",
					user: userhash.Hash{1},
					id:   "2",
					t:    3,
					tups: []brain.Tuple{
						{Prefix: strings.Fields("喜多 虹夏 リョウ ぼっち"), Suffix: ""},
						{Prefix: strings.Fields("虹夏 リョウ ぼっち"), Suffix: "喜多"},
						{Prefix: strings.Fields("リョウ ぼっち"), Suffix: "虹夏"},
						{Prefix: strings.Fields("ぼっち"), Suffix: "リョウ"},
						{Prefix: nil, Suffix: "ぼっち"},
					},
				},
			},
			know: []know{
				{
					tag:    "結束",
					id:     "2",
					prefix: "喜多\x00虹夏\x00リョウ\x00ぼっち\x00\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "虹夏\x00リョウ\x00ぼっち\x00\x00",
					suffix: "喜多",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "リョウ\x00ぼっち\x00\x00",
					suffix: "虹夏",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "ぼっち\x00\x00",
					suffix: "リョウ",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "\x00",
					suffix: "ぼっち",
				},
			},
			msgs: []msg{
				{
					tag:  "結束",
					id:   "2",
					time: 3e6,
					user: userhash.Hash{1},
				},
			},
		},
		{
			name: "msgs",
			learn: []learn{
				{
					tag:  "kessoku",
					user: userhash.Hash{1},
					id:   "2",
					t:    3,
					tups: []brain.Tuple{
						{Prefix: []string{"bocchi"}, Suffix: ""},
						{Prefix: nil, Suffix: "bocchi"},
					},
				},
				{
					tag:  "kessoku",
					user: userhash.Hash{4},
					id:   "5",
					t:    6,
					tups: []brain.Tuple{
						{Prefix: []string{"ryo"}, Suffix: ""},
						{Prefix: nil, Suffix: "ryo"},
					},
				},
			},
			know: []know{
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "bocchi\x00\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "\x00",
					suffix: "bocchi",
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "ryo\x00\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "\x00",
					suffix: "ryo",
				},
			},
			msgs: []msg{
				{
					tag:  "kessoku",
					id:   "2",
					time: 3e6,
					user: userhash.Hash{1},
				},
				{
					tag:  "kessoku",
					id:   "5",
					time: 6e6,
					user: userhash.Hash{4},
				},
			},
		},
		{
			name: "tagged",
			learn: []learn{
				{
					tag:  "kessoku",
					user: userhash.Hash{1},
					id:   "2",
					t:    3,
					tups: []brain.Tuple{
						{Prefix: []string{"bocchi"}, Suffix: ""},
						{Prefix: nil, Suffix: "bocchi"},
					},
				},
				{
					tag:  "sickhack",
					user: userhash.Hash{1},
					id:   "2",
					t:    3,
					tups: []brain.Tuple{
						{Prefix: []string{"kikuri"}, Suffix: ""},
						{Prefix: nil, Suffix: "kikuri"},
					},
				},
			},
			know: []know{
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "bocchi\x00\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "\x00",
					suffix: "bocchi",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "kikuri\x00\x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "\x00",
					suffix: "kikuri",
				},
			},
			msgs: []msg{
				{
					tag:  "kessoku",
					id:   "2",
					time: 3e6,
					user: userhash.Hash{1},
				},
				{
					tag:  "sickhack",
					id:   "2",
					time: 3e6,
					user: userhash.Hash{1},
				},
			},
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
			for _, m := range c.learn {
				msg := brain.Message{
					ID:        m.id,
					Sender:    m.user,
					Timestamp: m.t,
				}
				err := br.Learn(ctx, m.tag, &msg, m.tups)
				if err != nil {
					t.Errorf("failed to learn %v/%v: %v", m.tag, m.id, err)
				}
			}
			conn, err := db.Take(ctx)
			defer db.Put(conn)
			if err != nil {
				t.Fatalf("couldn't get conn to check db state: %v", err)
			}
			contents(t, conn, c.know, c.msgs)
		})
	}
}

func BenchmarkLearn(b *testing.B) {
	dir := filepath.ToSlash(b.TempDir())
	new := func(ctx context.Context, b *testing.B) brain.Learner {
		dsn := fmt.Sprintf("file:%s/benchmark_learn.db?_journal=WAL", dir)
		db, err := sqlitex.NewPool(dsn, sqlitex.PoolOptions{PrepareConn: sqlbrain.RecommendedPrep})
		if err != nil {
			b.Fatal(err)
		}
		{
			conn, err := db.Take(ctx)
			if err != nil {
				b.Fatal(err)
			}
			db.Put(conn)
		}
		br, err := sqlbrain.Open(ctx, db)
		if err != nil {
			b.Fatalf("couldn't open brain: %v", err)
		}
		return br
	}
	braintest.BenchLearn(context.Background(), b, new, func(l brain.Learner) { l.(*sqlbrain.Brain).Close() })
}
