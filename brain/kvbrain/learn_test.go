package kvbrain

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/braintest"
	"github.com/zephyrtronium/robot/userhash"
)

func mkey(tag, toks, id string) string {
	b := make([]byte, 0, 8+len(toks)+len(id))
	b = hashTag(b, tag)
	b = append(b, toks...)
	b = append(b, id...)
	return string(b)
}

func dbcheck(t *testing.T, db *badger.DB, want map[string]string) {
	t.Helper()
	seen := 0
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.IteratorOptions{}
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := string(item.Key())
			v, err := item.ValueCopy(nil)
			if err != nil {
				t.Errorf("couldn't get value for key %q: %v", k, err)
			}
			if got := string(v); want[k] != got {
				t.Errorf("wrong value for key %q: want %q, got %q", k, want[k], got)
			}
			seen++
		}
		return nil
	})
	if err != nil {
		t.Errorf("view failed: %v", err)
	}
	if seen != len(want) {
		t.Errorf("saw wrong number of items: want %d, got %d", len(want), seen)
	}
}

func TestLearn(t *testing.T) {
	uu := ":)"
	h := userhash.Hash{2}
	cases := []struct {
		name string
		id   string
		user userhash.Hash
		tag  string
		time time.Time
		tups []brain.Tuple
		want map[string]string
	}{
		{
			name: "single",
			id:   uu,
			user: h,
			tag:  "kessoku",
			time: time.Unix(0, 0),
			tups: []brain.Tuple{
				{
					Prefix: nil,
					Suffix: "bocchi",
				},
			},
			want: map[string]string{
				mkey("kessoku", "\xff", uu): "bocchi",
			},
		},
		{
			name: "full",
			id:   uu,
			user: h,
			tag:  "kessoku",
			time: time.Unix(0, 0),
			tups: []brain.Tuple{
				{
					Prefix: []string{"seika", "kita", "nijika", "ryou", "bocchi"},
					Suffix: "",
				},
				{
					Prefix: []string{"kita", "nijika", "ryou", "bocchi"},
					Suffix: "seika",
				},
				{
					Prefix: []string{"nijika", "ryou", "bocchi"},
					Suffix: "kita",
				},
				{
					Prefix: []string{"ryou", "bocchi"},
					Suffix: "nijika",
				},
				{
					Prefix: []string{"bocchi"},
					Suffix: "ryou",
				},
				{
					Prefix: nil,
					Suffix: "bocchi",
				},
			},
			want: map[string]string{
				mkey("kessoku", "\xff", uu):                                              "bocchi",
				mkey("kessoku", "bocchi\xff\xff", uu):                                    "ryou",
				mkey("kessoku", "ryou\xffbocchi\xff\xff", uu):                            "nijika",
				mkey("kessoku", "nijika\xffryou\xffbocchi\xff\xff", uu):                  "kita",
				mkey("kessoku", "kita\xffnijika\xffryou\xffbocchi\xff\xff", uu):          "seika",
				mkey("kessoku", "seika\xffkita\xffnijika\xffryou\xffbocchi\xff\xff", uu): "",
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
			msg := brain.Message{
				ID:        c.id,
				Sender:    c.user,
				Timestamp: c.time.UnixMilli(),
			}
			if err := br.Learn(ctx, c.tag, &msg, c.tups); err != nil {
				t.Errorf("failed to learn: %v", err)
			}
			dbcheck(t, db, c.want)
		})
	}
}

func BenchmarkLearn(b *testing.B) {
	new := func(ctx context.Context, b *testing.B) brain.Interface {
		db, err := badger.Open(badger.DefaultOptions(b.TempDir()).WithLogger(nil))
		if err != nil {
			b.Fatal(err)
		}
		return New(db)
	}
	cleanup := func(l brain.Interface) {
		br := l.(*Brain)
		if err := br.knowledge.DropAll(); err != nil {
			b.Fatal(err)
		}
		if err := br.knowledge.Close(); err != nil {
			b.Fatal(err)
		}
	}
	braintest.BenchLearn(context.Background(), b, new, cleanup)
}
