package sqlbrain_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/zephyrtronium/robot/v2/brain"
	"github.com/zephyrtronium/robot/v2/brain/sqlbrain"
	"gitlab.com/zephyrtronium/sq"
	"golang.org/x/exp/slices"
)

func TestForget(t *testing.T) {
	type insert struct {
		tag    string
		tuples []brain.Tuple
	}
	cases := []struct {
		name   string
		order  int
		insert []insert
		forget []insert
		left   []insert
	}{
		{
			name:   "empty-1",
			order:  1,
			insert: nil,
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			left: nil,
		},
		{
			name:  "success-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			left: nil,
		},
		{
			name:  "prefix-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "c"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "c"},
					},
				},
			},
		},
		{
			name:  "suffix-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"c"}, Suffix: "b"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
		},
		{
			name:  "single-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
		},
		{
			name:  "idempotent-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
						{Prefix: []string{"a"}, Suffix: "c"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "c"},
					},
				},
			},
		},
		{
			name:  "repeat-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
						{Prefix: []string{"a"}, Suffix: "b"},
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
		},
		{
			name:  "tagged-1",
			order: 1,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			forget: []insert{
				{
					tag: "homura",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
		},
		{
			name:   "empty-2",
			order:  2,
			insert: nil,
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			left: nil,
		},
		{
			name:  "success-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			left: nil,
		},
		{
			name:  "prefix-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "d"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "d"},
					},
				},
			},
		},
		{
			name:  "suffix-first-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"d", "b"}, Suffix: "c"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
		},
		{
			name:  "suffix-second-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "d"}, Suffix: "c"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
		},
		{
			name:  "single-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
		},
		{
			name:  "idempotent-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
						{Prefix: []string{"a", "b"}, Suffix: "d"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "d"},
					},
				},
			},
		},
		{
			name:  "repeat-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
						{Prefix: []string{"a", "b"}, Suffix: "c"},
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			forget: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
		},
		{
			name:  "tagged-2",
			order: 2,
			insert: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			forget: []insert{
				{
					tag: "homura",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			left: []insert{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
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
				// Double-check that the tuples are in.
				// This is largely testing that tuples() works as advertised.
				tups, err := tuples(ctx, db, v.tag, c.order)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(v.tuples, tups, cmpopts.EquateEmpty()); diff != "" {
					t.Fatalf("wrong tuples added before test (+got/-want):\n%s", diff)
				}
			}
			for i, v := range c.forget {
				err := br.Forget(ctx, v.tag, v.tuples)
				if err != nil {
					t.Errorf("couldn't forget %q (group %d): %v", v, i, err)
				}
			}
			var wantTags []string
			for _, left := range c.left {
				wantTags = append(wantTags, left.tag)
				tups, err := tuples(ctx, db, left.tag, c.order)
				if err != nil {
					t.Errorf("couldn't get remaining tuples for tag %s: %v", left.tag, err)
					continue
				}
				if diff := cmp.Diff(left.tuples, tups, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("wrong tuples left with tag %s (+got/-want):\n%s", left.tag, diff)
				}
			}
			sort.Strings(wantTags)
			gotTags, err := tags(ctx, t, db)
			if err != nil {
				t.Errorf("couldn't get tags list: %v", err)
			}
			if diff := cmp.Diff(wantTags, gotTags); diff != "" {
				t.Errorf("wrong tags have tuples (+got/-want):\n%s", diff)
			}
			if t.Failed() {
				dumpdb(ctx, t, db)
			}
		})
	}
}

func TestForgetMessage(t *testing.T) {
	type insert struct {
		id     uuid.UUID
		tag    string
		tuples []brain.Tuple
	}
	type remain struct {
		tag    string
		tuples []brain.Tuple
	}
	uuids := []uuid.UUID{
		uuid.New(),
		uuid.New(),
	}
	cases := []struct {
		name   string
		order  int
		insert []insert
		forget []uuid.UUID
		left   []remain
		errs   bool
	}{
		{
			name:  "single-1",
			order: 1,
			insert: []insert{
				{
					id:  uuids[0],
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			forget: []uuid.UUID{uuids[0]},
			left:   nil,
		},
		{
			name:  "multi-1",
			order: 1,
			insert: []insert{
				{
					id:  uuids[0],
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
						{Prefix: []string{"b"}, Suffix: "c"},
						{Prefix: []string{"c"}, Suffix: "d"},
					},
				},
			},
			forget: []uuid.UUID{uuids[0]},
			left:   nil,
		},
		{
			name:  "unmatched-1",
			order: 1,
			insert: []insert{
				{
					id:  uuids[0],
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			forget: []uuid.UUID{uuids[1]},
			left: []remain{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			errs: true,
		},
		{
			name:  "single-2",
			order: 2,
			insert: []insert{
				{
					id:  uuids[0],
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			forget: []uuid.UUID{uuids[0]},
			left:   nil,
		},
		{
			name:  "multi-2",
			order: 2,
			insert: []insert{
				{
					id:  uuids[0],
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
						{Prefix: []string{"b", "c"}, Suffix: "d"},
						{Prefix: []string{"c", "d"}, Suffix: "e"},
					},
				},
			},
			forget: []uuid.UUID{uuids[0]},
			left:   nil,
		},
		{
			name:  "unmatched-2",
			order: 2,
			insert: []insert{
				{
					id:  uuids[0],
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			forget: []uuid.UUID{uuids[1]},
			left: []remain{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			errs: true,
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
				md := brain.MessageMeta{
					ID:  v.id,
					Tag: v.tag,
				}
				err := addTuples(ctx, db, md, v.tuples)
				if err != nil {
					t.Fatal(err)
				}
				// Double-check that the tuples are in.
				tups, err := tuples(ctx, db, v.tag, c.order)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(v.tuples, tups, cmpopts.EquateEmpty()); diff != "" {
					t.Fatalf("wrong tuples added before test (+got/-want):\n%s", diff)
				}
			}
			for _, id := range c.forget {
				err := br.ForgetMessage(ctx, id)
				if err != nil && !c.errs {
					t.Errorf("couldn't forget message %v: %v", id, err)
				} else if err == nil && c.errs {
					t.Error("expected forget to fail")
				}
			}
			var wantTags []string
			for _, left := range c.left {
				wantTags = append(wantTags, left.tag)
				tups, err := tuples(ctx, db, left.tag, c.order)
				if err != nil {
					t.Errorf("couldn't get remaining tuples for tag %s: %v", left.tag, err)
					continue
				}
				if diff := cmp.Diff(left.tuples, tups, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("wrong tuples left with tag %s (+got/-want):\n%s", left.tag, diff)
				}
			}
			sort.Strings(wantTags)
			gotTags, err := tags(ctx, t, db)
			if err != nil {
				t.Errorf("couldn't get tags list: %v", err)
			}
			if diff := cmp.Diff(wantTags, gotTags); diff != "" {
				t.Errorf("wrong tags have tuples (+got/-want):\n%s", diff)
			}
			if t.Failed() {
				dumpdb(ctx, t, db)
			}
		})
	}
}

func TestForgetDuring(t *testing.T) {
	type insert struct {
		tag    string
		time   int64
		tuples []brain.Tuple
	}
	type remain struct {
		tag    string
		tuples []brain.Tuple
	}
	cases := []struct {
		name   string
		order  int
		insert []insert
		tag    string
		forget [2]int64
		left   []remain
	}{
		{
			name:  "single-1",
			order: 1,
			insert: []insert{
				{
					tag:  "madoka",
					time: 2,
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			tag:    "madoka",
			forget: [2]int64{1, 3},
			left:   nil,
		},
		{
			name:  "multiple-1",
			order: 1,
			insert: []insert{
				{
					tag:  "madoka",
					time: 2,
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
						{Prefix: []string{"b"}, Suffix: "c"},
					},
				},
			},
			tag:    "madoka",
			forget: [2]int64{1, 3},
			left:   nil,
		},
		{
			name:  "outside-1",
			order: 1,
			insert: []insert{
				{
					tag:  "madoka",
					time: 4,
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			tag:    "madoka",
			forget: [2]int64{1, 3},
			left: []remain{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
		},
		{
			name:  "tagged-1",
			order: 1,
			insert: []insert{
				{
					tag:  "madoka",
					time: 2,
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
			tag:    "homura",
			forget: [2]int64{1, 3},
			left: []remain{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a"}, Suffix: "b"},
					},
				},
			},
		},
		{
			name:  "single-2",
			order: 2,
			insert: []insert{
				{
					tag:  "madoka",
					time: 2,
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			tag:    "madoka",
			forget: [2]int64{1, 3},
			left:   nil,
		},
		{
			name:  "multiple-2",
			order: 2,
			insert: []insert{
				{
					tag:  "madoka",
					time: 2,
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
						{Prefix: []string{"b", "c"}, Suffix: "d"},
					},
				},
			},
			tag:    "madoka",
			forget: [2]int64{1, 3},
			left:   nil,
		},
		{
			name:  "outside-2",
			order: 2,
			insert: []insert{
				{
					tag:  "madoka",
					time: 4,
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			tag:    "madoka",
			forget: [2]int64{1, 3},
			left: []remain{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
		},
		{
			name:  "tagged-2",
			order: 2,
			insert: []insert{
				{
					tag:  "madoka",
					time: 2,
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
			},
			tag:    "homura",
			forget: [2]int64{1, 3},
			left: []remain{
				{
					tag: "madoka",
					tuples: []brain.Tuple{
						{Prefix: []string{"a", "b"}, Suffix: "c"},
					},
				},
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
				md := brain.MessageMeta{
					ID:   uuid.New(),
					Time: time.UnixMilli(v.time),
					Tag:  v.tag,
				}
				err := addTuples(ctx, db, md, v.tuples)
				if err != nil {
					t.Fatal(err)
				}
				// Double-check that the tuples are in.
				tups, err := tuples(ctx, db, v.tag, c.order)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(v.tuples, tups, cmpopts.EquateEmpty()); diff != "" {
					t.Fatalf("wrong tuples added before test (+got/-want):\n%s", diff)
				}
			}
			a, b := time.UnixMilli(c.forget[0]), time.UnixMilli(c.forget[1])
			if err := br.ForgetDuring(ctx, c.tag, a, b); err != nil {
				t.Errorf("could't forget: %v", err)
			}
			var wantTags []string
			for _, left := range c.left {
				wantTags = append(wantTags, left.tag)
				tups, err := tuples(ctx, db, left.tag, c.order)
				if err != nil {
					t.Errorf("couldn't get remaining tuples for tag %s: %v", left.tag, err)
					continue
				}
				if diff := cmp.Diff(left.tuples, tups, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("wrong tuples left with tag %s (+got/-want):\n%s", left.tag, diff)
				}
			}
			sort.Strings(wantTags)
			gotTags, err := tags(ctx, t, db)
			if err != nil {
				t.Errorf("couldn't get tags list: %v", err)
			}
			if diff := cmp.Diff(wantTags, gotTags); diff != "" {
				t.Errorf("wrong tags have tuples (+got/-want):\n%s", diff)
			}
			if t.Failed() {
				dumpdb(ctx, t, db)
			}
		})
	}
}

// tuples gets all tuples in db with the given tag. The returned tuples are in
// lexicographically ascending order.
func tuples(ctx context.Context, db sqlbrain.DB, tag string, order int) ([]brain.Tuple, error) {
	rows, err := db.Query(ctx, "SELECT Tuple.* FROM Tuple JOIN Message AS m ON m.id = Tuple.msg WHERE m.tag = ? AND m.deleted IS NULL", tag)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var all []brain.Tuple
	r := make([]any, order+2)
	for i := range r {
		r[i] = &sq.NullString{}
	}
	for rows.Next() {
		if err := rows.Scan(r...); err != nil {
			panic(err)
		}
		t := brain.Tuple{Prefix: make([]string, order)}
		for i := range t.Prefix {
			s := r[i+1].(*sq.NullString)
			if s.Valid {
				t.Prefix[i] = s.String
			}
		}
		s := r[order+1].(*sq.NullString)
		if s.Valid {
			t.Suffix = s.String
		}
		all = append(all, t)
	}
	sort.Slice(all, func(i, j int) bool {
		switch d := slices.Compare(all[i].Prefix, all[j].Prefix); d {
		case -1:
			return true
		case 1:
			return false
		default:
			return all[i].Suffix < all[j].Suffix
		}
	})
	return all, rows.Err()
}

// tags gets all tags with any associated tuples in db in ascending order.
func tags(ctx context.Context, t *testing.T, db sqlbrain.DB) ([]string, error) {
	t.Helper()
	rows, err := db.Query(ctx, `SELECT DISTINCT m.tag FROM Message AS m INNER JOIN Tuple ON m.id = Tuple.msg WHERE m.deleted IS NULL ORDER BY tag`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			panic(err)
		}
		tags = append(tags, s)
	}
	return tags, rows.Err()
}
