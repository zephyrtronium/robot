package kvbrain

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

func mkey(tag, toks string, id uuid.UUID) string {
	b := make([]byte, 8, 8+len(toks)+len(id))
	binary.LittleEndian.PutUint64(b, hashTag(tag))
	b = append(b, toks...)
	b = append(b, id[:]...)
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
	uu := uuid.UUID{':', ')', ':', ')', ':', ')', ':', ')', ':', ')', ':', ')', ':', ')', ':', ')'}
	h := userhash.Hash{2}
	cases := []struct {
		name string
		msg  brain.MessageMeta
		tups []brain.Tuple
		want map[string]string
	}{
		{
			name: "single",
			msg: brain.MessageMeta{
				ID:   uu,
				User: h,
				Tag:  "kessoku",
				Time: time.Unix(0, 0),
			},
			tups: []brain.Tuple{
				{
					Prefix: []string{""},
					Suffix: "bocchi",
				},
			},
			want: map[string]string{
				mkey("kessoku", "\xff", uu): "bocchi",
			},
		},
		{
			name: "full",
			msg: brain.MessageMeta{
				ID:   uu,
				User: h,
				Tag:  "kessoku",
				Time: time.Unix(0, 0),
			},
			tups: []brain.Tuple{
				{
					Prefix: []string{"", "", "", ""},
					Suffix: "bocchi",
				},
				{
					Prefix: []string{"", "", "", "bocchi"},
					Suffix: "ryou",
				},
				{
					Prefix: []string{"", "", "bocchi", "ryou"},
					Suffix: "nijika",
				},
				{
					Prefix: []string{"", "bocchi", "ryou", "nijika"},
					Suffix: "kita",
				},
				{
					Prefix: []string{"bocchi", "ryou", "nijika", "kita"},
					Suffix: "seika",
				},
				{
					Prefix: []string{"ryou", "nijika", "kita", "seika"},
					Suffix: "",
				},
			},
			want: map[string]string{
				mkey("kessoku", "\xff", uu):                                     "bocchi",
				mkey("kessoku", "bocchi\xff\xff", uu):                           "ryou",
				mkey("kessoku", "ryou\xffbocchi\xff\xff", uu):                   "nijika",
				mkey("kessoku", "nijika\xffryou\xffbocchi\xff\xff", uu):         "kita",
				mkey("kessoku", "kita\xffnijika\xffryou\xffbocchi\xff\xff", uu): "seika",
				mkey("kessoku", "seika\xffkita\xffnijika\xffryou\xff\xff", uu):  "",
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
			if err := br.Learn(ctx, &c.msg, c.tups); err != nil {
				t.Errorf("failed to learn: %v", err)
			}
			dbcheck(t, db, c.want)
		})
	}
}
