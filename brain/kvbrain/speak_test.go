package kvbrain

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

func TestSpeak(t *testing.T) {
	uu := uuid.UUID{':', ')', ':', ')', ':', ')', ':', ')', ':', ')', ':', ')', ':', ')', ':', ')'}
	cases := []struct {
		name   string
		kvs    [][2]string
		prompt []string
		want   [][]string
	}{
		{
			name: "empty",
			kvs:  nil,
			want: [][]string{
				// Even with no thoughts head empty, we expect to get empty,
				// non-error results when we speak. Our test currently records
				// what it gets as a joined string for convenience, so we want
				// an empty string in here, even though we really should be
				// getting an empty slice.
				{""},
			},
		},
		{
			name: "single",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", uu), "bocchi"},
				{mkey("kessoku", "bocchi\xff\xff", uu), ""},
			},
			want: [][]string{
				{"bocchi"},
			},
		},
		{
			name: "longer",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", uu), "bocchi"},
				{mkey("kessoku", "bocchi\xff\xff", uu), "ryou"},
				{mkey("kessoku", "ryou\xffbocchi\xff\xff", uu), "nijika"},
				{mkey("kessoku", "nijika\xffryou\xffbocchi\xff\xff", uu), "kita"},
				{mkey("kessoku", "kita\xffnijika\xffryou\xffbocchi\xff\xff", uu), ""},
			},
			want: [][]string{
				{"bocchi", "ryou", "nijika", "kita"},
			},
		},
		{
			name: "entropy",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", uu), "BOCCHI"},
				{mkey("kessoku", "bocchi\xff\xff", uu), "RYOU"},
				{mkey("kessoku", "ryou\xffbocchi\xff\xff", uu), "NIJIKA"},
				{mkey("kessoku", "nijika\xffryou\xffbocchi\xff\xff", uu), "KITA"},
				{mkey("kessoku", "kita\xffnijika\xffryou\xffbocchi\xff\xff", uu), ""},
			},
			want: [][]string{
				{"BOCCHI", "RYOU", "NIJIKA", "KITA"},
			},
		},
		{
			name: "prompted",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", uu), "bocchi"},
				{mkey("kessoku", "bocchi\xff\xff", uu), "ryou"},
				{mkey("kessoku", "ryou\xffbocchi\xff\xff", uu), "nijika"},
				{mkey("kessoku", "nijika\xffryou\xffbocchi\xff\xff", uu), "kita"},
				{mkey("kessoku", "kita\xffnijika\xffryou\xffbocchi\xff\xff", uu), ""},
			},
			prompt: []string{"bocchi"},
			want: [][]string{
				{"bocchi", "ryou", "nijika", "kita"},
			},
		},
		{
			name: "prompted-entropy",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", uu), "BOCCHI"},
				{mkey("kessoku", "bocchi\xff\xff", uu), "RYOU"},
				{mkey("kessoku", "ryou\xffbocchi\xff\xff", uu), "NIJIKA"},
				{mkey("kessoku", "nijika\xffryou\xffbocchi\xff\xff", uu), "KITA"},
				{mkey("kessoku", "kita\xffnijika\xffryou\xffbocchi\xff\xff", uu), ""},
			},
			prompt: []string{"BOCCHI"},
			want: [][]string{
				{"BOCCHI", "RYOU", "NIJIKA", "KITA"},
			},
		},
		{
			name: "uniform",
			kvs: [][2]string{
				{mkey("kessoku", "\xff", uuid.UUID{1}), "bocchi"},
				{mkey("kessoku", "bocchi\xff\xff", uuid.UUID{1}), "ryou"},
				{mkey("kessoku", "ryou\xffbocchi\xff\xff", uuid.UUID{1}), ""},
				{mkey("kessoku", "\xff", uuid.UUID{2}), "bocchi"},
				{mkey("kessoku", "bocchi\xff\xff", uuid.UUID{2}), "nijika"},
				{mkey("kessoku", "nijika\xffbocchi\xff\xff", uuid.UUID{2}), ""},
				{mkey("kessoku", "\xff", uuid.UUID{3}), "bocchi"},
				{mkey("kessoku", "bocchi\xff\xff", uuid.UUID{3}), "kita"},
				{mkey("kessoku", "kita\xffbocchi\xff\xff", uuid.UUID{3}), ""},
			},
			want: [][]string{
				{"bocchi", "ryou"},
				{"bocchi", "nijika"},
				{"bocchi", "kita"},
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
				want[strings.Join(v, ":")] = true
			}
			got := make(map[string]bool, len(c.want))
			for range 256 {
				m, err := br.Speak(ctx, "kessoku", slices.Clone(c.prompt))
				if err != nil {
					t.Errorf("failed to speak: %v", err)
				}
				got[strings.Join(m, ":")] = true
			}
			if !maps.Equal(want, got) {
				t.Errorf("wrong results: want %v, got %v", want, got)
			}
		})
	}
}
