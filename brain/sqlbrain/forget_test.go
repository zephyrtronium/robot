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
		{
			tag:  "結束",
			user: userhash.Hash{4},
			id:   "5",
			t:    6,
			tups: []brain.Tuple{
				{Prefix: []string{"bocchi"}, Suffix: ""},
				{Prefix: nil, Suffix: "bocchi"},
			},
		},
		{
			tag:  "結束",
			user: userhash.Hash{7},
			id:   "8",
			t:    9,
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
				{Prefix: []string{"bocchi"}, Suffix: ""},
				{Prefix: nil, Suffix: "bocchi"},
			},
		},
	}
	initKnow := []know{
		{
			tag:    "結束",
			id:     "2",
			prefix: "喜多\x00虹夏\x00リョウ\x00ぼっち\x00",
			suffix: "",
		},
		{
			tag:    "結束",
			id:     "2",
			prefix: "虹夏\x00リョウ\x00ぼっち\x00",
			suffix: "喜多",
		},
		{
			tag:    "結束",
			id:     "2",
			prefix: "リョウ\x00ぼっち\x00",
			suffix: "虹夏",
		},
		{
			tag:    "結束",
			id:     "2",
			prefix: "ぼっち\x00",
			suffix: "リョウ",
		},
		{
			tag:    "結束",
			id:     "2",
			prefix: "",
			suffix: "ぼっち",
		},
		{
			tag:    "結束",
			id:     "5",
			prefix: "bocchi\x00",
			suffix: "",
		},
		{
			tag:    "結束",
			id:     "5",
			prefix: "",
			suffix: "bocchi",
		},
		{
			tag:    "結束",
			id:     "8",
			prefix: "bocchi\x00",
			suffix: "",
		},
		{
			tag:    "結束",
			id:     "8",
			prefix: "",
			suffix: "bocchi",
		},
		{
			tag:    "sickhack",
			id:     "2",
			prefix: "bocchi\x00",
			suffix: "",
		},
		{
			tag:    "sickhack",
			id:     "2",
			prefix: "",
			suffix: "bocchi",
		},
	}
	initMsgs := []msg{
		{
			tag:  "結束",
			id:   "2",
			time: 3,
			user: userhash.Hash{1},
		},
		{
			tag:  "結束",
			id:   "5",
			time: 6,
			user: userhash.Hash{4},
		},
		{
			tag:  "結束",
			id:   "8",
			time: 9,
			user: userhash.Hash{7},
		},
		{
			tag:  "sickhack",
			id:   "2",
			time: 3,
			user: userhash.Hash{1},
		},
	}
	type forget struct {
		tag  string
		tups []brain.Tuple
	}
	cases := []struct {
		name   string
		forget []forget
		know   []know
		msgs   []msg
	}{
		{
			name:   "empty",
			forget: nil,
			know:   initKnow,
			msgs:   initMsgs,
		},
		{
			name: "none",
			forget: []forget{
				{
					tag: "結束",
					tups: []brain.Tuple{
						{Prefix: strings.Fields("tuples such no"), Suffix: ""},
						{Prefix: strings.Fields("such no"), Suffix: "tuples"},
						{Prefix: strings.Fields("no"), Suffix: "such"},
						{Prefix: nil, Suffix: "no"},
					},
				},
			},
			know: initKnow,
			msgs: initMsgs,
		},
		{
			name: "single",
			forget: []forget{
				{
					tag: "結束",
					tups: []brain.Tuple{
						{Prefix: nil, Suffix: "ぼっち"},
					},
				},
			},
			know: []know{
				{
					tag:    "結束",
					id:     "2",
					prefix: "喜多\x00虹夏\x00リョウ\x00ぼっち\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "虹夏\x00リョウ\x00ぼっち\x00",
					suffix: "喜多",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "リョウ\x00ぼっち\x00",
					suffix: "虹夏",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "ぼっち\x00",
					suffix: "リョウ",
				},
				{
					tag:     "結束",
					id:      "2",
					prefix:  "",
					suffix:  "ぼっち",
					deleted: ref("FORGET"),
				},
				{
					tag:    "結束",
					id:     "5",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "5",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:    "結束",
					id:     "8",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "8",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "",
					suffix: "bocchi",
				},
			},
			msgs: initMsgs,
		},
		{
			name: "all",
			forget: []forget{
				{
					tag: "結束",
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
					tag:     "結束",
					id:      "2",
					prefix:  "喜多\x00虹夏\x00リョウ\x00ぼっち\x00",
					suffix:  "",
					deleted: ref("FORGET"),
				},
				{
					tag:     "結束",
					id:      "2",
					prefix:  "虹夏\x00リョウ\x00ぼっち\x00",
					suffix:  "喜多",
					deleted: ref("FORGET"),
				},
				{
					tag:     "結束",
					id:      "2",
					prefix:  "リョウ\x00ぼっち\x00",
					suffix:  "虹夏",
					deleted: ref("FORGET"),
				},
				{
					tag:     "結束",
					id:      "2",
					prefix:  "ぼっち\x00",
					suffix:  "リョウ",
					deleted: ref("FORGET"),
				},
				{
					tag:     "結束",
					id:      "2",
					prefix:  "",
					suffix:  "ぼっち",
					deleted: ref("FORGET"),
				},
				{
					tag:    "結束",
					id:     "5",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "5",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:    "結束",
					id:     "8",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "8",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "",
					suffix: "bocchi",
				},
			},
			msgs: initMsgs,
		},
		{
			name: "once",
			forget: []forget{
				{
					tag: "結束",
					tups: []brain.Tuple{
						{Prefix: nil, Suffix: "bocchi"},
					},
				},
			},
			know: []know{
				{
					tag:    "結束",
					id:     "2",
					prefix: "喜多\x00虹夏\x00リョウ\x00ぼっち\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "虹夏\x00リョウ\x00ぼっち\x00",
					suffix: "喜多",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "リョウ\x00ぼっち\x00",
					suffix: "虹夏",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "ぼっち\x00",
					suffix: "リョウ",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "",
					suffix: "ぼっち",
				},
				{
					tag:    "結束",
					id:     "5",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:     "結束",
					id:      "5",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("FORGET"),
				},
				{
					tag:    "結束",
					id:     "8",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "8",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "",
					suffix: "bocchi",
				},
			},
			msgs: initMsgs,
		},
		{
			name: "multi",
			forget: []forget{
				{
					tag: "結束",
					tups: []brain.Tuple{
						{Prefix: nil, Suffix: "bocchi"},
					},
				},
				{
					tag: "結束",
					tups: []brain.Tuple{
						{Prefix: nil, Suffix: "bocchi"},
					},
				},
			},
			know: []know{
				{
					tag:    "結束",
					id:     "2",
					prefix: "喜多\x00虹夏\x00リョウ\x00ぼっち\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "虹夏\x00リョウ\x00ぼっち\x00",
					suffix: "喜多",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "リョウ\x00ぼっち\x00",
					suffix: "虹夏",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "ぼっち\x00",
					suffix: "リョウ",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "",
					suffix: "ぼっち",
				},
				{
					tag:    "結束",
					id:     "5",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:     "結束",
					id:      "5",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("FORGET"),
				},
				{
					tag:    "結束",
					id:     "8",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:     "結束",
					id:      "8",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("FORGET"),
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "",
					suffix: "bocchi",
				},
			},
			msgs: initMsgs,
		},
		{
			name: "tag",
			forget: []forget{
				{
					tag: "sickhack",
					tups: []brain.Tuple{
						{Prefix: nil, Suffix: "bocchi"},
					},
				},
			},
			know: []know{
				{
					tag:    "結束",
					id:     "2",
					prefix: "喜多\x00虹夏\x00リョウ\x00ぼっち\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "虹夏\x00リョウ\x00ぼっち\x00",
					suffix: "喜多",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "リョウ\x00ぼっち\x00",
					suffix: "虹夏",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "ぼっち\x00",
					suffix: "リョウ",
				},
				{
					tag:    "結束",
					id:     "2",
					prefix: "",
					suffix: "ぼっち",
				},
				{
					tag:    "結束",
					id:     "5",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "5",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:    "結束",
					id:     "8",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "結束",
					id:     "8",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:     "sickhack",
					id:      "2",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("FORGET"),
				},
			},
			msgs: initMsgs,
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
			for _, v := range c.forget {
				err := br.Forget(ctx, v.tag, v.tups)
				if err != nil {
					t.Errorf("couldn't forget %q in %v: %v", v.tups, v.tag, err)
				}
			}
			contents(t, conn, c.know, c.msgs)
		})
	}
}

func TestForgetMessage(t *testing.T) {
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
			prefix: "kita\x00nijika\x00ryo\x00bocchi\x00",
			suffix: "",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "nijika\x00ryo\x00bocchi\x00",
			suffix: "kita",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "ryo\x00bocchi\x00",
			suffix: "nijika",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "bocchi\x00",
			suffix: "ryo",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "",
			suffix: "bocchi",
		},
		{
			tag:    "kessoku",
			id:     "5",
			prefix: "bocchi\x00",
			suffix: "",
		},
		{
			tag:    "kessoku",
			id:     "5",
			prefix: "",
			suffix: "bocchi",
		},
		{
			tag:    "sickhack",
			id:     "2",
			prefix: "kikuri\x00",
			suffix: "",
		},
		{
			tag:    "sickhack",
			id:     "2",
			prefix: "",
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
					prefix:  "kita\x00nijika\x00ryo\x00bocchi\x00",
					suffix:  "",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "nijika\x00ryo\x00bocchi\x00",
					suffix:  "kita",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "ryo\x00bocchi\x00",
					suffix:  "nijika",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "bocchi\x00",
					suffix:  "ryo",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "kikuri\x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "",
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
					prefix: "kita\x00nijika\x00ryo\x00bocchi\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "nijika\x00ryo\x00bocchi\x00",
					suffix: "kita",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "ryo\x00bocchi\x00",
					suffix: "nijika",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "bocchi\x00",
					suffix: "ryo",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:     "kessoku",
					id:      "5",
					prefix:  "bocchi\x00",
					suffix:  "",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "kessoku",
					id:      "5",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "kikuri\x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "",
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
					prefix: "kita\x00nijika\x00ryo\x00bocchi\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "nijika\x00ryo\x00bocchi\x00",
					suffix: "kita",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "ryo\x00bocchi\x00",
					suffix: "nijika",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "bocchi\x00",
					suffix: "ryo",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:     "sickhack",
					id:      "2",
					prefix:  "kikuri\x00",
					suffix:  "",
					deleted: ref("CLEARMSG"),
				},
				{
					tag:     "sickhack",
					id:      "2",
					prefix:  "",
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
			if err := br.ForgetMessage(ctx, c.tag, c.id); err != nil {
				t.Errorf("failed to delete %v/%v: %v", c.tag, c.id, err)
			}
			contents(t, conn, c.know, c.msgs)
		})
	}
}

func TestForgetDuring(t *testing.T) {
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
			prefix: "kita\x00nijika\x00ryo\x00bocchi\x00",
			suffix: "",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "nijika\x00ryo\x00bocchi\x00",
			suffix: "kita",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "ryo\x00bocchi\x00",
			suffix: "nijika",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "bocchi\x00",
			suffix: "ryo",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "",
			suffix: "bocchi",
		},
		{
			tag:    "kessoku",
			id:     "5",
			prefix: "bocchi\x00",
			suffix: "",
		},
		{
			tag:    "kessoku",
			id:     "5",
			prefix: "",
			suffix: "bocchi",
		},
		{
			tag:    "sickhack",
			id:     "2",
			prefix: "kikuri\x00",
			suffix: "",
		},
		{
			tag:    "sickhack",
			id:     "2",
			prefix: "",
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
		name   string
		tag    string
		since  int64
		before int64
		know   []know
		msgs   []msg
	}{
		{
			name:   "none",
			tag:    "kessoku",
			since:  100,
			before: 200,
			know:   initKnow,
			msgs:   initMsgs,
		},
		{
			name:   "early",
			tag:    "kessoku",
			since:  1,
			before: 4,
			know: []know{
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "kita\x00nijika\x00ryo\x00bocchi\x00",
					suffix:  "",
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "nijika\x00ryo\x00bocchi\x00",
					suffix:  "kita",
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "ryo\x00bocchi\x00",
					suffix:  "nijika",
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "bocchi\x00",
					suffix:  "ryo",
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("TIME"),
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "kikuri\x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "",
					suffix: "kikuri",
				},
			},
			msgs: []msg{
				{
					tag:     "kessoku",
					id:      "2",
					time:    3,
					user:    userhash.Hash{1},
					deleted: ref("TIME"),
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
			name:   "late",
			tag:    "kessoku",
			since:  5,
			before: 8,
			know: []know{
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "kita\x00nijika\x00ryo\x00bocchi\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "nijika\x00ryo\x00bocchi\x00",
					suffix: "kita",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "ryo\x00bocchi\x00",
					suffix: "nijika",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "bocchi\x00",
					suffix: "ryo",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:     "kessoku",
					id:      "5",
					prefix:  "bocchi\x00",
					suffix:  "",
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "5",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("TIME"),
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "kikuri\x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "",
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
					deleted: ref("TIME"),
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
			name:   "all",
			tag:    "kessoku",
			since:  1,
			before: 8,
			know: []know{
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "kita\x00nijika\x00ryo\x00bocchi\x00",
					suffix:  "",
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "nijika\x00ryo\x00bocchi\x00",
					suffix:  "kita",
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "ryo\x00bocchi\x00",
					suffix:  "nijika",
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "bocchi\x00",
					suffix:  "ryo",
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "5",
					prefix:  "bocchi\x00",
					suffix:  "",
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "5",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("TIME"),
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "kikuri\x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "",
					suffix: "kikuri",
				},
			},
			msgs: []msg{
				{
					tag:     "kessoku",
					id:      "2",
					time:    3,
					user:    userhash.Hash{1},
					deleted: ref("TIME"),
				},
				{
					tag:     "kessoku",
					id:      "5",
					time:    6,
					user:    userhash.Hash{4},
					deleted: ref("TIME"),
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
			name:   "tagged",
			tag:    "sickhack",
			since:  1,
			before: 7,
			know: []know{
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "kita\x00nijika\x00ryo\x00bocchi\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "nijika\x00ryo\x00bocchi\x00",
					suffix: "kita",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "ryo\x00bocchi\x00",
					suffix: "nijika",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "bocchi\x00",
					suffix: "ryo",
				},
				{
					tag:    "kessoku",
					id:     "2",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "bocchi\x00",
					suffix: "",
				},
				{
					tag:    "kessoku",
					id:     "5",
					prefix: "",
					suffix: "bocchi",
				},
				{
					tag:     "sickhack",
					id:      "2",
					prefix:  "kikuri\x00",
					suffix:  "",
					deleted: ref("TIME"),
				},
				{
					tag:     "sickhack",
					id:      "2",
					prefix:  "",
					suffix:  "kikuri",
					deleted: ref("TIME"),
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
					deleted: ref("TIME"),
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
			since, before := time.Unix(0, c.since), time.Unix(0, c.before)
			if err := br.ForgetDuring(ctx, c.tag, since, before); err != nil {
				t.Errorf("couldn't delete in %v between %d and %d: %v", c.tag, c.since, c.before, err)
			}
			contents(t, conn, c.know, c.msgs)
		})
	}
}

func TestForgetUser(t *testing.T) {
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
			user: userhash.Hash{1},
			id:   "5",
			t:    6,
			tups: []brain.Tuple{
				{Prefix: []string{"bocchi"}, Suffix: ""},
				{Prefix: nil, Suffix: "bocchi"},
			},
		},
		{
			tag:  "sickhack",
			user: userhash.Hash{4},
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
			prefix: "kita\x00nijika\x00ryo\x00bocchi\x00",
			suffix: "",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "nijika\x00ryo\x00bocchi\x00",
			suffix: "kita",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "ryo\x00bocchi\x00",
			suffix: "nijika",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "bocchi\x00",
			suffix: "ryo",
		},
		{
			tag:    "kessoku",
			id:     "2",
			prefix: "",
			suffix: "bocchi",
		},
		{
			tag:    "kessoku",
			id:     "5",
			prefix: "bocchi\x00",
			suffix: "",
		},
		{
			tag:    "kessoku",
			id:     "5",
			prefix: "",
			suffix: "bocchi",
		},
		{
			tag:    "sickhack",
			id:     "2",
			prefix: "kikuri\x00",
			suffix: "",
		},
		{
			tag:    "sickhack",
			id:     "2",
			prefix: "",
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
			user: userhash.Hash{1},
		},
		{
			tag:  "sickhack",
			id:   "2",
			time: 3,
			user: userhash.Hash{4},
		},
	}
	cases := []struct {
		name string
		user userhash.Hash
		know []know
		msgs []msg
	}{
		{
			name: "none",
			user: userhash.Hash{100},
			know: initKnow,
			msgs: initMsgs,
		},
		{
			name: "all",
			user: userhash.Hash{1},
			know: []know{
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "kita\x00nijika\x00ryo\x00bocchi\x00",
					suffix:  "",
					deleted: ref("CLEARCHAT"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "nijika\x00ryo\x00bocchi\x00",
					suffix:  "kita",
					deleted: ref("CLEARCHAT"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "ryo\x00bocchi\x00",
					suffix:  "nijika",
					deleted: ref("CLEARCHAT"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "bocchi\x00",
					suffix:  "ryo",
					deleted: ref("CLEARCHAT"),
				},
				{
					tag:     "kessoku",
					id:      "2",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("CLEARCHAT"),
				},
				{
					tag:     "kessoku",
					id:      "5",
					prefix:  "bocchi\x00",
					suffix:  "",
					deleted: ref("CLEARCHAT"),
				},
				{
					tag:     "kessoku",
					id:      "5",
					prefix:  "",
					suffix:  "bocchi",
					deleted: ref("CLEARCHAT"),
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "kikuri\x00",
					suffix: "",
				},
				{
					tag:    "sickhack",
					id:     "2",
					prefix: "",
					suffix: "kikuri",
				},
			},
			msgs: []msg{
				{
					tag:     "kessoku",
					id:      "2",
					time:    3,
					user:    userhash.Hash{1},
					deleted: ref("CLEARCHAT"),
				},
				{
					tag:     "kessoku",
					id:      "5",
					time:    6,
					user:    userhash.Hash{1},
					deleted: ref("CLEARCHAT"),
				},
				{
					tag:  "sickhack",
					id:   "2",
					time: 3,
					user: userhash.Hash{4},
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
			if err := br.ForgetUser(ctx, &c.user); err != nil {
				t.Errorf("couldn't delete from %x: %v", c.user, err)
			}
			contents(t, conn, c.know, c.msgs)
		})
	}
}
