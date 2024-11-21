package sqlbrain_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/sqlbrain"
	"github.com/zephyrtronium/robot/userhash"
)

func TestForget(t *testing.T) {
	learn := []learn{
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
		{
			tag:  "kessoku",
			user: userhash.Hash{4},
			id:   "5",
			t:    6,
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
	}
	initKnow := []know{
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
		{
			tag:    "kessoku",
			id:     "5",
			prefix: "bocchi\x00\x00",
			suffix: "",
		},
		{
			tag:    "kessoku",
			id:     "5",
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
	}
	initMsgs := []msg{
		{
			tag:  "kessoku",
			id:   "2",
			time: 3,
			user: userhash.Hash{1},
		},
		{
			tag:  "kessoku",
			id:   "5",
			time: 6,
			user: userhash.Hash{4},
		},
		{
			tag:  "sickhack",
			id:   "2",
			time: 3,
			user: userhash.Hash{1},
		},
	}
	cases := []struct {
		name string
		tag  string
		id   string
		know []know
		msgs []msg
	}{
		{
			name: "none",
			tag:  "kessoku",
			id:   "",
			know: initKnow,
			msgs: initMsgs,
		},
		{
			name: "first",
			tag:  "kessoku",
			id:   "2",
			know: []know{
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "kita\x00nijika\x00ryo\x00bocchi\x00\x00",
					suffix:  "",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "nijika\x00ryo\x00bocchi\x00\x00",
					suffix:  "kita",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "ryo\x00bocchi\x00\x00",
					suffix:  "nijika",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "bocchi\x00\x00",
					suffix:  "ryo",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "\x00",
					suffix:  "bocchi",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "bocchi\x00\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "5",
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
					tag:     "kessoku",
					id:      "2",
					time:    3,
					user:    userhash.Hash{1},
					deleted: ref("CLEARMSG"),
				},
				{
					tag:  "kessoku",
					id:   "5",
					time: 6,
					user: userhash.Hash{4},
				},
				{
					tag:  "sickhack",
					id:   "2",
					time: 3,
					user: userhash.Hash{1},
				},
			},
		},
		{
			name: "second",
			tag:  "kessoku",
			id:   "5",
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
				{
					tag:     "kessoku",
					id:      "5",
					prefix:  "bocchi\x00\x00",
					suffix:  "",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "kessoku",
					id:      "5",
					prefix:  "\x00",
					suffix:  "bocchi",
					deleted: ref("CLEARMSG"),
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
					time: 3,
					user: userhash.Hash{1},
				},
				{
					tag:     "kessoku",
					id:      "5",
					time:    6,
					user:    userhash.Hash{4},
					deleted: ref("CLEARMSG"),
				},
				{
					tag:  "sickhack",
					id:   "2",
					time: 3,
					user: userhash.Hash{1},
				},
			},
		},
		{
			name: "tagged",
			tag:  "sickhack",
			id:   "2",
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
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "bocchi\x00\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "\x00",
					suffix: "bocchi",
				},
				{
					tag:     "sickhack",
					id:      "2",
					prefix:  "kikuri\x00\x00",
					suffix:  "",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "sickhack",
					id:      "2",
					prefix:  "\x00",
					suffix:  "kikuri",
					deleted: ref("CLEARMSG"),
				},
			},
			msgs: []msg{
				{
					tag:  "kessoku",
					id:   "2",
					time: 3,
					user: userhash.Hash{1},
				},
				{
					tag:  "kessoku",
					id:   "5",
					time: 6,
					user: userhash.Hash{4},
				},
				{
					tag:     "sickhack",
					id:      "2",
					time:    3,
					user:    userhash.Hash{1},
					deleted: ref("CLEARMSG"),
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
			for _, m := range learn {
				err := br.Learn(ctx, m.tag, m.id, m.user, time.Unix(0, m.t), m.tups)
				if err != nil {
					t.Errorf("failed to learn %v/%v: %v", m.tag, m.id, err)
				}
			}
			conn, err := db.Take(ctx)
			defer db.Put(conn)
			if err != nil {
				t.Fatalf("couldn't get conn to check db state: %v", err)
			}
			contents(t, conn, initKnow, initMsgs)
			if t.Failed() {
				t.Fatal("setup failed")
			}
			if err := br.Forget(ctx, c.tag, c.id); err != nil {
				t.Errorf("failed to delete %v/%v: %v", c.tag, c.id, err)
			}
			contents(t, conn, c.know, c.msgs)
		})
	}
}
