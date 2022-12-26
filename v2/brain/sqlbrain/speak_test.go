package sqlbrain_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/zephyrtronium/robot/v2/brain"
	"github.com/zephyrtronium/robot/v2/brain/sqlbrain"
	"gitlab.com/zephyrtronium/sq"
	"golang.org/x/exp/slices"
)

func TestNew(t *testing.T) {
	type insert struct {
		tag    string
		tuples []brain.Tuple
	}
	cases := []struct {
		name   string
		order  int
		insert []insert
		tag    string
		want   [][]string
	}{
		{
			name:  "include-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{""}, Suffix: "a"},
						{Prefix: []string{""}, Suffix: "b"},
					},
				},
			},
			tag: "madoka",
			want: [][]string{
				{"a"},
				{"b"},
			},
		},
		{
			name:  "start-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{""}, Suffix: "a"},
						{Prefix: []string{""}, Suffix: "b"},
						{Prefix: []string{"b"}, Suffix: "c"},
					},
				},
			},
			tag: "madoka",
			want: [][]string{
				{"a"},
				{"b"},
			},
		},
		{
			name:  "tagged-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{""}, Suffix: "a"},
						{Prefix: []string{""}, Suffix: "b"},
					},
				},
				{
					tag: "homura",
					tuples: []brain.Tuple{
						{Prefix: []string{""}, Suffix: "x"},
						{Prefix: []string{""}, Suffix: "y"},
					},
				},
			},
			tag: "madoka",
			want: [][]string{
				{"a"},
				{"b"},
			},
		},
		{
			name:  "include-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"", ""}, Suffix: "a"},
						{Prefix: []string{"", ""}, Suffix: "b"},
					},
				},
			},
			tag: "madoka",
			want: [][]string{
				{"", "a"},
				{"", "b"},
			},
		},
		{
			name:  "start-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"", ""}, Suffix: "a"},
						{Prefix: []string{"", ""}, Suffix: "b"},
						{Prefix: []string{"", "b"}, Suffix: "c"},
					},
				},
			},
			tag: "madoka",
			want: [][]string{
				{"", "a"},
				{"", "b"},
			},
		},
		{
			name:  "tagged-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"", ""}, Suffix: "a"},
						{Prefix: []string{"", ""}, Suffix: "b"},
					},
				},
				{
					tag: "homura",
					tuples: []brain.Tuple{
						{Prefix: []string{"", ""}, Suffix: "x"},
						{Prefix: []string{"", ""}, Suffix: "y"},
					},
				},
			},
			tag: "madoka",
			want: [][]string{
				{"", "a"},
				{"", "b"},
			},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := testDB(c.order)
			br, err := sqlbrain.Open(ctx, db)
			if err != nil {
				t.Fatalf("couldn't open brain: %v", err)
			}
			for _, v := range c.insert {
				err := addTuples(ctx, db, tagged(v.tag), v.tuples)
				if err != nil {
					t.Fatal(err)
				}
			}
			var got [][]string
			for i := 0; i < 100; i++ {
				p, err := br.New(ctx, c.tag)
				if err != nil {
					t.Errorf("err from new: %v", err)
				}
				got = lexset(got, p)
			}
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("wrong prompts: (-want/+got)\n%s", diff)
			}
			if t.Failed() {
				dumpdb(ctx, t, db)
			}
		})
	}
}

func TestSpeak(t *testing.T) {
	type insert struct {
		tag    string
		tuples []brain.Tuple
	}
	cases := []struct {
		name   string
		order  int
		insert []insert
		tag    string
		prompt []string
		want   [][]string
	}{
		{
			name:  "include-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{""}, Suffix: "a"},
						{Prefix: []string{"a"}, Suffix: "b"},
						{Prefix: []string{"b"}, Suffix: ""},
					},
				},
			},
			tag:    "madoka",
			prompt: []string{""},
			want: [][]string{
				{"a", "b"},
			},
		},
		{
			name:  "branch-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{""}, Suffix: "a"},
						{Prefix: []string{"a"}, Suffix: "b"},
						{Prefix: []string{"a"}, Suffix: "c"},
						{Prefix: []string{"b"}, Suffix: ""},
						{Prefix: []string{"c"}, Suffix: ""},
					},
				},
			},
			tag:    "madoka",
			prompt: []string{""},
			want: [][]string{
				{"a", "b"},
				{"a", "c"},
			},
		},
		{
			name:  "tagged-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{""}, Suffix: "a"},
						{Prefix: []string{"a"}, Suffix: "b"},
						{Prefix: []string{"b"}, Suffix: ""},
					},
				},
				{
					tag: "homura",
					tuples: []brain.Tuple{
						{Prefix: []string{""}, Suffix: "a"},
						{Prefix: []string{"a"}, Suffix: "c"},
						{Prefix: []string{"c"}, Suffix: ""},
					},
				},
			},
			tag:    "madoka",
			prompt: []string{""},
			want: [][]string{
				{"a", "b"},
			},
		},
		{
			name:  "include-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"", ""}, Suffix: "a"},
						{Prefix: []string{"", "a"}, Suffix: "b"},
						{Prefix: []string{"a", "b"}, Suffix: ""},
					},
				},
			},
			tag:    "madoka",
			prompt: []string{"", ""},
			want: [][]string{
				{"a", "b"},
			},
		},
		{
			name:  "branch-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"", ""}, Suffix: "a"},
						{Prefix: []string{"", "a"}, Suffix: "b"},
						{Prefix: []string{"", "a"}, Suffix: "c"},
						{Prefix: []string{"a", "b"}, Suffix: ""},
						{Prefix: []string{"a", "c"}, Suffix: ""},
					},
				},
			},
			tag:    "madoka",
			prompt: []string{"", ""},
			want: [][]string{
				{"a", "b"},
				{"a", "c"},
			},
		},
		{
			name:  "tagged-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"", ""}, Suffix: "a"},
						{Prefix: []string{"", "a"}, Suffix: "b"},
						{Prefix: []string{"a", "b"}, Suffix: ""},
					},
				},
				{
					tag: "homura",
					tuples: []brain.Tuple{
						{Prefix: []string{"", ""}, Suffix: "a"},
						{Prefix: []string{"", "a"}, Suffix: "c"},
						{Prefix: []string{"a", "c"}, Suffix: ""},
					},
				},
			},
			tag:    "madoka",
			prompt: []string{"", ""},
			want: [][]string{
				{"a", "b"},
			},
		},
		{
			name:  "long",
			order: 4,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"", "", "", ""}, Suffix: "a"},
						{Prefix: []string{"", "", "", "a"}, Suffix: "b"},
						{Prefix: []string{"", "", "a", "b"}, Suffix: "c"},
						{Prefix: []string{"", "a", "b", "c"}, Suffix: "d"},
						{Prefix: []string{"a", "b", "c", "d"}, Suffix: "e"},
						{Prefix: []string{"b", "c", "d", "e"}, Suffix: "f"},
						{Prefix: []string{"c", "d", "e", "f"}, Suffix: ""},
					},
				},
			},
			tag:    "madoka",
			prompt: []string{"", "", "", ""},
			want: [][]string{
				{"a", "b", "c", "d", "e", "f"},
			},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := testDB(c.order)
			br, err := sqlbrain.Open(ctx, db)
			if err != nil {
				t.Fatalf("couldn't open brain: %v", err)
			}
			for _, v := range c.insert {
				err := addTuples(ctx, db, tagged(v.tag), v.tuples)
				if err != nil {
					t.Fatal(err)
				}
			}
			var got [][]string
			for i := 0; i < 100; i++ {
				msg, err := br.Speak(ctx, c.tag, c.prompt)
				if err != nil {
					t.Errorf("err from speak: %v", err)
				}
				got = lexset(got, trim(msg))
			}
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("wrong prompts: (-want/+got)\n%s", diff)
			}
			if t.Failed() {
				dumpdb(ctx, t, db)
			}
		})
	}
}

// addTuples inserts tuples into a test db.
// TODO(zeph): use user & time metadata
func addTuples(ctx context.Context, db sqlbrain.DB, msg brain.MessageMeta, tuples []brain.Tuple) error {
	order := len(tuples[0].Prefix)
	tx, err := db.Begin(ctx, nil)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback()
	_, err = tx.Exec(ctx, "INSERT INTO Message(id, user, tag, time) VALUES (?, x'', ?, ?)", msg.ID, msg.Tag, msg.Time.UnixMilli())
	if err != nil {
		return fmt.Errorf("couldn't add message: %v", err)
	}
	var b strings.Builder
	for _, tup := range tuples {
		b.Reset()
		b.WriteString("INSERT INTO Tuple(msg")
		for i := 0; i < order; i++ {
			fmt.Fprintf(&b, ", p%d", i)
		}
		b.WriteString(", suffix) VALUES (?, ?")
		b.WriteString(strings.Repeat(", ?", order))
		b.WriteByte(')')
		args := []any{msg.ID}
		for _, w := range tup.Prefix {
			args = append(args, sq.NullString{String: w, Valid: w != ""})
		}
		args = append(args, sq.NullString{String: tup.Suffix, Valid: tup.Suffix != ""})
		_, err := tx.Exec(ctx, b.String(), args...)
		if err != nil {
			return fmt.Errorf("couldn't add tuples %q with query %q: %w", tuples, b.String(), err)
		}
	}
	return tx.Commit()
}

// tagged is a shortcut to create message metadata holding only a given tag
// and a random UUID.
func tagged(tag string) brain.MessageMeta {
	return brain.MessageMeta{
		ID:  uuid.New(),
		Tag: tag,
	}
}

// lexset adds a []string to a [][]string such that the latter remains in
// sorted order without duplicates.
func lexset(dst [][]string, n []string) [][]string {
	k, ok := slices.BinarySearchFunc(dst, n, slices.Compare[string])
	if ok {
		return dst
	}
	return slices.Insert(dst, k, n)
}

func trim(r []string) []string {
	for k := len(r) - 1; k >= 0; k-- {
		if r[k] != "" {
			r = r[:k+1]
			break
		}
	}
	for k, v := range r {
		if v != "" {
			return r[k:]
		}
	}
	return nil
}

func dumpdb(ctx context.Context, t *testing.T, db sqlbrain.DB) {
	t.Helper()
	t.Log("db content:")
	rows, err := db.Query(ctx, "SELECT m.user, m.tag, m.time, m.deleted, Tuple.* FROM Message AS m JOIN Tuple ON m.id = Tuple.msg")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		panic(err)
	}
	t.Log(cols)
	for rows.Next() {
		r := make([]any, len(cols))
		for i := range r {
			r[i] = &r[i]
		}
		if err := rows.Scan(r...); err != nil {
			panic(err)
		}
		t.Logf("%q", r)
	}
	if rows.Err() != nil {
		t.Log(rows.Err())
	}
}
