package kvbrain

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/braintest"
)

func TestThink(t *testing.T) {
	cases := []struct {
		name   string
		kvs    [][2]string
		tag    string
		prompt []string
		want   []string
	}{
		{
			name:   "empty",
			kvs:    nil,
			tag:    "kessoku",
			prompt: nil,
			want:   nil,
		},
		{
			name: "empty-tagged",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", "1"), "bocchi "},
				{mkey("kessoku", "bocchi \xff\xff", "1"), ""},
			},
			tag:    "sickhack",
			prompt: nil,
			want:   nil,
		},
		{
			name: "empty-prompted",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", "1"), "bocchi "},
				{mkey("kessoku", "bocchi \xff\xff", "1"), ""},
			},
			tag:    "kessoku",
			prompt: []string{"kikuri "},
			want:   nil,
		},
		{
			name: "single",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", "1"), "bocchi "},
				{mkey("kessoku", "bocchi \xff\xff", "1"), ""},
			},
			tag:    "kessoku",
			prompt: nil,
			want:   []string{"bocchi "},
		},
		{
			name: "several",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", "1"), "bocchi "},
				{mkey("kessoku", "bocchi \xff\xff", "1"), "ryo "},
				{mkey("kessoku", "ryo \xffbocchi \xff\xff", "1"), "nijika "},
				{mkey("kessoku", "kita \xffryo \xffbocchi \xff\xff", "1"), ""},
			},
			tag:    "kessoku",
			prompt: nil,
			want:   []string{"bocchi "},
		},
		{
			name: "multi",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", "1"), "member "},
				{mkey("kessoku", "member \xff\xff", "1"), "bocchi "},
				{mkey("kessoku", "bocchi \xffmember \xff\xff", "1"), ""},
				{mkey("kessoku", "\xff", "2"), "member "},
				{mkey("kessoku", "member \xff\xff", "2"), "ryo "},
				{mkey("kessoku", "ryo \xffmember \xff\xff", "2"), ""},
				{mkey("kessoku", "\xff", "3"), "member "},
				{mkey("kessoku", "member \xff\xff", "3"), "nijika "},
				{mkey("kessoku", "nijika \xffmember \xff\xff", "3"), ""},
				{mkey("kessoku", "\xff", "4"), "member "},
				{mkey("kessoku", "member \xff\xff", "4"), "kita "},
				{mkey("kessoku", "kita \xffmember \xff\xff", "4"), ""},
			},
			tag:    "kessoku",
			prompt: nil,
			want:   []string{"member ", "member ", "member ", "member "},
		},
		{
			name: "multi-prompted",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", "1"), "member "},
				{mkey("kessoku", "member \xff\xff", "1"), "bocchi "},
				{mkey("kessoku", "bocchi \xffmember \xff\xff", "1"), ""},
				{mkey("kessoku", "\xff", "2"), "member "},
				{mkey("kessoku", "member \xff\xff", "2"), "ryo "},
				{mkey("kessoku", "ryo \xffmember \xff\xff", "2"), ""},
				{mkey("kessoku", "\xff", "3"), "member "},
				{mkey("kessoku", "member \xff\xff", "3"), "nijika "},
				{mkey("kessoku", "nijika \xffmember \xff\xff", "3"), ""},
				{mkey("kessoku", "\xff", "4"), "member "},
				{mkey("kessoku", "member \xff\xff", "4"), "kita "},
				{mkey("kessoku", "kita \xffmember \xff\xff", "4"), ""},
			},
			tag:    "kessoku",
			prompt: []string{"member "},
			want:   []string{"bocchi ", "ryo ", "nijika ", "kita "},
		},
		{
			name: "multi-tagged",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", "1"), "member "},
				{mkey("kessoku", "member \xff\xff", "1"), "bocchi "},
				{mkey("kessoku", "bocchi \xffmember \xff\xff", "1"), ""},
				{mkey("kessoku", "\xff", "2"), "member "},
				{mkey("kessoku", "member \xff\xff", "2"), "ryo "},
				{mkey("kessoku", "ryo \xffmember \xff\xff", "2"), ""},
				{mkey("kessoku", "\xff", "3"), "member "},
				{mkey("kessoku", "member \xff\xff", "3"), "nijika "},
				{mkey("kessoku", "nijika \xffmember \xff\xff", "3"), ""},
				{mkey("kessoku", "\xff", "4"), "member "},
				{mkey("kessoku", "member \xff\xff", "4"), "kita "},
				{mkey("kessoku", "kita \xffmember \xff\xff", "4"), ""},
				{mkey("sickhack", "\xff", "5"), "member "},
				{mkey("sickhack", "member \xff\xff", "5"), "kikuri "},
				{mkey("sickhack", "kikuri \xffmember \xff\xff", "5"), ""},
				{mkey("sickhack", "\xff", "6"), "member "},
				{mkey("sickhack", "member \xff\xff", "6"), "eliza "},
				{mkey("sickhack", "eliza \xffmember \xff\xff", "6"), ""},
				{mkey("sickhack", "\xff", "7"), "member "},
				{mkey("sickhack", "member \xff\xff", "7"), "shima "},
				{mkey("sickhack", "shima \xffmember \xff\xff", "7"), ""},
			},
			tag:    "sickhack",
			prompt: nil,
			want:   []string{"member ", "member ", "member "},
		},
		// TODO(zeph): forgotten terms; these aren't implemented correctly yet
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil))
			if err != nil {
				t.Fatal(err)
			}
			err = db.Update(func(txn *badger.Txn) error {
				var err error
				for _, item := range c.kvs {
					err = errors.Join(err, txn.Set([]byte(item[0]), []byte(item[1])))
				}
				return err
			})
			if err != nil {
				t.Errorf("couldn't set up knowledge: %v", err)
			}
			br := New(db)
			_, got, err := braintest.Collect(br.Think(ctx, c.tag, c.prompt))
			if err != nil {
				t.Errorf("couldn't think: %v", err)
			}
			slices.Sort(c.want)
			slices.Sort(got)
			if !slices.Equal(c.want, got) {
				t.Errorf("wrong results:\nwant %q\ngot  %q", c.want, got)
			}
		})
	}
}

func BenchmarkSpeak(b *testing.B) {
	new := func(ctx context.Context, b *testing.B) brain.Interface {
		db, err := badger.Open(badger.DefaultOptions(b.TempDir()).WithLogger(nil).WithCompression(options.None).WithBloomFalsePositive(1.0 / 32).WithNumMemtables(16).WithLevelSizeMultiplier(4))
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
	braintest.BenchSpeak(context.Background(), b, new, cleanup)
}
