package spoken_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-json-experiment/json"
	"github.com/google/go-cmp/cmp"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/zephyrtronium/robot/spoken"
)

var dbCount atomic.Int64

func testDB() *sqlitex.Pool {
	k := dbCount.Add(1)
	pool, err := sqlitex.NewPool(fmt.Sprintf("file:test-record-%d.db?mode=memory&cache=shared", k), sqlitex.PoolOptions{Flags: sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenMemory | sqlite.OpenSharedCache | sqlite.OpenURI})
	if err != nil {
		panic(err)
	}
	return pool
}

func TestRecord(t *testing.T) {
	ctx := context.Background()
	db := testDB()
	conn, err := db.Take(ctx)
	defer db.Put(conn)
	if err != nil {
		t.Fatalf("couldn't get conn: %v", err)
	}
	h, err := spoken.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	err = h.Record(ctx, "kessoku", "boccho ryo xD", []string{"1", "2"}, time.Unix(1, 0), time.Second, "bocchi ryo", "xD", "o")
	if err != nil {
		t.Errorf("couldn't record: %v", err)
	}

	opts := sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			tag := stmt.ColumnText(0)
			msg := stmt.ColumnText(1)
			trace := stmt.ColumnText(2)
			tm := stmt.ColumnInt64(3)
			meta := stmt.ColumnText(4)

			if tag != "kessoku" {
				t.Errorf("wrong tag recorded: want %q, got %q", "kessoku", tag)
			}
			if msg != "boccho ryo xD" {
				t.Errorf("wrong message recorded: want %q, got %q", "bocchi ryo", msg)
			}
			var tr []string
			if err := json.Unmarshal([]byte(trace), &tr); err != nil {
				t.Errorf("couldn't unmarshal trace from %q: %v", trace, err)
			}
			if !slices.Equal(tr, []string{"1", "2"}) {
				t.Errorf("wrong trace recorded: want %q, got %q from %q", []string{"1", "2"}, tr, trace)
			}
			if got, want := time.Unix(0, tm), time.Unix(1, 0); got != want {
				t.Errorf("wrong time: want %v, got %v", want, got)
			}
			var md map[string]any
			if err := json.Unmarshal([]byte(meta), &md); err != nil {
				t.Errorf("couldn't unmarshal metadata from %q: %v", meta, md)
			}
			want := map[string]any{
				"orig":   "bocchi ryo",
				"emote":  "xD",
				"effect": "o",
				"cost":   float64(time.Second.Nanoseconds()),
			}
			if !maps.Equal(md, want) {
				t.Errorf("wrong metadata recorded: want %v, got %v from %q", want, md, meta)
			}
			return nil
		},
	}
	err = sqlitex.ExecuteTransient(conn, `SELECT tag, msg, JSON(trace), time, JSON(meta) FROM spoken`, &opts)
	if err != nil {
		t.Errorf("failed to scan: %v", err)
	}
}

func TestTrace(t *testing.T) {
	// Create test fixture first.
	ctx := context.Background()
	db := testDB()
	h, err := spoken.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	insert := []struct {
		tag   string
		msg   string
		trace string
		time  int64
	}{
		{"kessoku", "bocchi", `["1"]`, 1},
		{"kessoku", "ryo", `["2"]`, 2},
		{"sickhack", "bocchi", `["3"]`, 3},
		{"kessoku", "ryo", `["4"]`, 4},
	}
	{
		conn, err := db.Take(ctx)
		if err != nil {
			t.Fatalf("couldn't get conn: %v", err)
		}
		st, err := conn.Prepare("INSERT INTO spoken (tag, msg, trace, time, meta) VALUES (:tag, :msg, JSONB(:trace), :time, JSONB('{}'))")
		if err != nil {
			t.Fatalf("couldn't prep insert: %v", err)
		}
		for _, r := range insert {
			st.SetText(":tag", r.tag)
			st.SetText(":msg", r.msg)
			st.SetText(":trace", r.trace)
			st.SetInt64(":time", r.time)
			_, err := st.Step()
			if err != nil {
				t.Errorf("failed to insert %v: %v", r, err)
			}
			if err := st.Reset(); err != nil {
				t.Errorf("couldn't reset: %v", err)
			}
		}
		if err := st.Finalize(); err != nil {
			t.Fatalf("couldn't finalize insert: %v", err)
		}
		db.Put(conn)
	}

	cases := []struct {
		name string
		tag  string
		msg  string
		want []string
		time time.Time
	}{
		{
			name: "none",
			tag:  "kessoku",
			msg:  "nijika",
			want: nil,
			time: time.Time{},
		},
		{
			name: "single",
			tag:  "kessoku",
			msg:  "bocchi",
			want: []string{"1"},
			time: time.Unix(0, 1),
		},
		{
			name: "latest",
			tag:  "kessoku",
			msg:  "ryo",
			want: []string{"4"},
			time: time.Unix(0, 4),
		},
		{
			name: "tagged",
			tag:  "sickhack",
			msg:  "bocchi",
			want: []string{"3"},
			time: time.Unix(0, 3),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			trace, tm, err := h.Trace(context.Background(), c.tag, c.msg)
			if err != nil {
				t.Errorf("couldn't get trace: %v", err)
			}
			if !slices.Equal(trace, c.want) {
				t.Errorf("wrong trace: want %q, got %q", c.want, trace)
			}
			if !tm.Equal(c.time) {
				t.Errorf("wrong time: want %v, got %v", c.time, tm.UnixNano())
			}
		})
	}
}

func TestSince(t *testing.T) {
	// Create test fixture first.
	ctx := context.Background()
	db := testDB()
	h, err := spoken.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	insert := []struct {
		tag   string
		msg   string
		trace string
		time  int64
	}{
		{"kessoku", "bocchi", `["1"]`, 10},
		{"kessoku", "ryo", `["2"]`, 20},
		{"sickhack", "bocchi", `["3"]`, 30},
		{"kessoku", "ryo", `["4"]`, 40},
	}
	{
		conn, err := db.Take(ctx)
		if err != nil {
			t.Fatalf("couldn't get conn: %v", err)
		}
		st, err := conn.Prepare("INSERT INTO spoken (tag, msg, trace, time, meta) VALUES (:tag, :msg, JSONB(:trace), :time, JSONB('{}'))")
		if err != nil {
			t.Fatalf("couldn't prep insert: %v", err)
		}
		for _, r := range insert {
			st.SetText(":tag", r.tag)
			st.SetText(":msg", r.msg)
			st.SetText(":trace", r.trace)
			st.SetInt64(":time", r.time)
			_, err := st.Step()
			if err != nil {
				t.Errorf("failed to insert %v: %v", r, err)
			}
			if err := st.Reset(); err != nil {
				t.Errorf("couldn't reset: %v", err)
			}
		}
		if err := st.Finalize(); err != nil {
			t.Fatalf("couldn't finalize insert: %v", err)
		}
		db.Put(conn)
	}

	cases := []struct {
		name string
		tag  string
		time int64
		want []string
	}{
		{
			name: "none",
			tag:  "kessoku",
			time: 1000,
			want: nil,
		},
		{
			name: "some",
			tag:  "kessoku",
			time: 15,
			want: []string{"2", "4"},
		},
		{
			name: "tagged",
			tag:  "sickhack",
			time: 15,
			want: []string{"3"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for id, err := range h.Since(ctx, c.tag, time.Unix(0, c.time)) {
				if err != nil {
					t.Error(err)
					continue
				}
				got = append(got, id)
			}
			slices.Sort(got)
			if !slices.Equal(c.want, got) {
				t.Errorf("wrong ids: want %q, got %q", c.want, got)
			}
		})
	}
}

func TestPrevious(t *testing.T) {
	// Create test fixture first.
	ctx := context.Background()
	db := testDB()
	h, err := spoken.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	insert := []struct {
		tag   string
		msg   string
		trace string
		time  int64
	}{
		{"kessoku", "bocchi", `["1"]`, 10},
		{"kessoku", "ryo", `["2"]`, 20},
		{"sickhack", "bocchi", `["3"]`, 30},
		{"kessoku", "ryo", `["4"]`, 40},
	}
	{
		conn, err := db.Take(ctx)
		if err != nil {
			t.Fatalf("couldn't get conn: %v", err)
		}
		st, err := conn.Prepare(`INSERT INTO spoken (tag, msg, trace, time, meta) VALUES (:tag, :msg, JSONB(:trace), :time, JSONB('{"cost":1000000000}'))`)
		if err != nil {
			t.Fatalf("couldn't prep insert: %v", err)
		}
		for _, r := range insert {
			st.SetText(":tag", r.tag)
			st.SetText(":msg", r.msg)
			st.SetText(":trace", r.trace)
			st.SetInt64(":time", r.time)
			_, err := st.Step()
			if err != nil {
				t.Errorf("failed to insert %v: %v", r, err)
			}
			if err := st.Reset(); err != nil {
				t.Errorf("couldn't reset: %v", err)
			}
		}
		if err := st.Finalize(); err != nil {
			t.Fatalf("couldn't finalize insert: %v", err)
		}
		db.Put(conn)
	}

	cases := []struct {
		name string
		tag  string
		n    int
		want []spoken.Message
	}{
		{
			name: "zero",
			tag:  "kessoku",
			n:    0,
			want: nil,
		},
		{
			name: "one",
			tag:  "kessoku",
			n:    1,
			want: []spoken.Message{
				{
					Text:  "ryo",
					Trace: []string{"4"},
					Time:  time.Unix(0, 40),
					Cost:  "1s",
				},
			},
		},
		{
			name: "ten",
			tag:  "kessoku",
			n:    10,
			want: []spoken.Message{
				{
					Text:  "ryo",
					Trace: []string{"4"},
					Time:  time.Unix(0, 40),
					Cost:  "1s",
				},
				{
					Text:  "ryo",
					Trace: []string{"2"},
					Time:  time.Unix(0, 20),
					Cost:  "1s",
				},
				{
					Text:  "bocchi",
					Trace: []string{"1"},
					Time:  time.Unix(0, 10),
					Cost:  "1s",
				},
			},
		},
		{
			name: "tagged",
			tag:  "sickhack",
			n:    10,
			want: []spoken.Message{
				{
					Text:  "bocchi",
					Trace: []string{"3"},
					Time:  time.Unix(0, 30),
					Cost:  "1s",
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			it, errf := h.Previous(t.Context(), c.tag, c.n)
			got := slices.Collect(it)
			if err := errf(); err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("wrong result (+got/-want):\n%s", diff)
			}
		})
	}
}
