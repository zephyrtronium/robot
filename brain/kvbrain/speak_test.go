package kvbrain

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/braintest"
)

func TestSpeak(t *testing.T) {
	uu := ":)"
	cases := []struct {
		name   string
		kvs    [][2]string
		prompt []string
		want   []string
	}{
		{
			name: "empty",
			kvs:  nil,
			want: []string{
				// Even with no thoughts head empty, we expect to get empty,
				// non-error results when we speak. Our test currently records
				// what it gets as a joined string for convenience, so we want
				// an empty string in here, even though we really should be
				// getting an empty slice.
				"",
			},
		},
		{
			name: "single",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", uu), "bocchi "},
				{mkey("kessoku", "bocchi \xff\xff", uu), ""},
			},
			want: []string{
				"bocchi ",
			},
		},
		{
			name: "longer",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", uu), "bocchi "},
				{mkey("kessoku", "bocchi \xff\xff", uu), "ryou "},
				{mkey("kessoku", "ryou \xffbocchi \xff\xff", uu), "nijika "},
				{mkey("kessoku", "nijika \xffryou \xffbocchi \xff\xff", uu), "kita "},
				{mkey("kessoku", "kita \xffnijika \xffryou \xffbocchi \xff\xff", uu), ""},
			},
			want: []string{
				"bocchi ryou nijika kita ",
			},
		},
		{
			name: "entropy",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", uu), "BOCCHI "},
				{mkey("kessoku", "bocchi \xff\xff", uu), "RYOU "},
				{mkey("kessoku", "ryou \xffbocchi \xff\xff", uu), "NIJIKA "},
				{mkey("kessoku", "nijika \xffryou \xffbocchi \xff\xff", uu), "KITA "},
				{mkey("kessoku", "kita \xffnijika \xffryou \xffbocchi \xff\xff", uu), ""},
			},
			want: []string{
				"BOCCHI RYOU NIJIKA KITA ",
			},
		},
		{
			name: "prompted",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", uu), "bocchi "},
				{mkey("kessoku", "bocchi \xff\xff", uu), "ryou "},
				{mkey("kessoku", "ryou \xffbocchi \xff\xff", uu), "nijika "},
				{mkey("kessoku", "nijika \xffryou \xffbocchi \xff\xff", uu), "kita "},
				{mkey("kessoku", "kita \xffnijika \xffryou \xffbocchi \xff\xff", uu), ""},
			},
			prompt: []string{"bocchi "},
			want: []string{
				"ryou nijika kita ",
			},
		},
		{
			name: "prompted-entropy",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", uu), "BOCCHI "},
				{mkey("kessoku", "bocchi \xff\xff", uu), "RYOU "},
				{mkey("kessoku", "ryou \xffbocchi \xff\xff", uu), "NIJIKA "},
				{mkey("kessoku", "nijika \xffryou \xffbocchi \xff\xff", uu), "KITA "},
				{mkey("kessoku", "kita \xffnijika \xffryou \xffbocchi \xff\xff", uu), ""},
			},
			prompt: []string{"bocchi "},
			want: []string{
				"RYOU NIJIKA KITA ",
			},
		},
		{
			name: "uniform",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", "1"), "bocchi "},
				{mkey("kessoku", "bocchi \xff\xff", "1"), "ryou "},
				{mkey("kessoku", "ryou \xffbocchi \xff\xff", "1"), ""},
				{mkey("kessoku", "\xff", "2"), "bocchi "},
				{mkey("kessoku", "bocchi \xff\xff", "2"), "nijika "},
				{mkey("kessoku", "nijika \xffbocchi \xff\xff", "2"), ""},
				{mkey("kessoku", "\xff", "3"), "bocchi "},
				{mkey("kessoku", "bocchi \xff\xff", "3"), "kita "},
				{mkey("kessoku", "kita \xffbocchi \xff\xff", "3"), ""},
			},
			want: []string{
				"bocchi ryou ",
				"bocchi nijika ",
				"bocchi kita ",
			},
		},
		// TODO(zeph): test tag isolation
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil))
			if err != nil {
				t.Fatal(err)
			}
			db.Update(func(txn *badger.Txn) error {
				var err error
				for _, item := range c.kvs {
					err = errors.Join(err, txn.Set([]byte(item[0]), []byte(item[1])))
				}
				return err
			})
			br := New(db)
			want := make(map[string]bool, len(c.want))
			for _, v := range c.want {
				want[v] = true
			}
			got := make(map[string]bool, len(c.want))
			var w brain.Builder
			for range 256 {
				w.Reset()
				err := br.Speak(ctx, "kessoku", slices.Clone(c.prompt), &w)
				if err != nil {
					t.Errorf("failed to speak: %v", err)
				}
				got[w.String()] = true
			}
			if !maps.Equal(want, got) {
				t.Errorf("wrong results: want %v, got %v", want, got)
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
