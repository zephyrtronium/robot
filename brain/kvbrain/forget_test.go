package kvbrain

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

func TestForget(t *testing.T) {
	type message struct {
		id   string
		user userhash.Hash
		tag  string
		time time.Time
		tups []brain.Tuple
	}
	cases := []struct {
		name string
		msgs []message
		uu   string
		want map[string]string
	}{
		{
			name: "single",
			msgs: []message{
				{
					id:   "1",
					user: userhash.Hash{2},
					tag:  "kessoku",
					time: time.Unix(0, 0),
					tups: []brain.Tuple{
						{Prefix: []string{"bocchi"}, Suffix: "ryou"},
					},
				},
			},
			uu: "1",
			want: map[string]string{
				mkey("kessoku", "bocchi\xff\xff", "1"): "ryou",
				mkey("kessoku", "\xfe\xfe", "1"):       "",
			},
		},
		{
			name: "several",
			msgs: []message{
				{
					id:   "1",
					user: userhash.Hash{2},
					tag:  "kessoku",
					time: time.Unix(0, 0),
					tups: []brain.Tuple{
						{Prefix: []string{"bocchi"}, Suffix: "ryou"},
						{Prefix: []string{"nijika"}, Suffix: "kita"},
					},
				},
			},
			uu: "1",
			want: map[string]string{
				mkey("kessoku", "bocchi\xff\xff", "1"): "ryou",
				mkey("kessoku", "nijika\xff\xff", "1"): "kita",
				mkey("kessoku", "\xfe\xfe", "1"):       "",
			},
		},
		{
			name: "tagged",
			msgs: []message{
				{
					id:   "1",
					user: userhash.Hash{2},
					tag:  "sickhack",
					time: time.Unix(0, 0),
					tups: []brain.Tuple{
						{Prefix: []string{"bocchi"}, Suffix: "ryou"},
					},
				},
			},
			uu: "1",
			want: map[string]string{
				mkey("sickhack", "bocchi\xff\xff", "1"): "ryou",
				mkey("kessoku", "\xfe\xfe", "1"):        "",
			},
		},
		{
			name: "unseen",
			msgs: []message{
				{
					id:   "1",
					user: userhash.Hash{2},
					tag:  "kessoku",
					time: time.Unix(0, 0),
					tups: []brain.Tuple{
						{Prefix: []string{"bocchi"}, Suffix: "ryou"},
					},
				},
			},
			uu: "2",
			want: map[string]string{
				mkey("kessoku", "bocchi\xff\xff", "1"): "ryou",
				mkey("kessoku", "\xfe\xfe", "2"):       "",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil))
			if err != nil {
				t.Fatal(err)
			}
			br := New(db)
			for _, msg := range c.msgs {
				m := brain.Message{
					ID:        msg.id,
					Sender:    msg.user,
					Timestamp: msg.time.UnixMilli(),
				}
				err := br.Learn(ctx, msg.tag, &m, msg.tups)
				if err != nil {
					t.Errorf("failed to learn: %v", err)
				}
			}
			if err := br.Forget(ctx, "kessoku", c.uu); err != nil {
				t.Errorf("couldn't forget: %v", err)
			}
			dbcheck(t, db, c.want)
		})
	}
}
