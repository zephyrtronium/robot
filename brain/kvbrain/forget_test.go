package kvbrain

import (
	"bytes"
	"context"
	"slices"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

func TestPastRecord(t *testing.T) {
	var p past
	ch := make(chan struct{})
	for i := range len(p.id) {
		go func() {
			p.record("1"+string(byte(i)), userhash.Hash{2, byte(i)}, int64(i), [][]byte{{4, byte(i)}, {5}})
			ch <- struct{}{}
		}()
	}
	for range len(p.id) {
		<-ch
	}
	if p.k != 0 {
		t.Errorf("wrong final index: %d should be zero", p.k)
	}
	for k := range len(p.id) {
		key, id, user, time := p.key[k], p.id[k], p.user[k], p.time[k]
		if want := [][]byte{{4, byte(time)}, {5}}; !slices.EqualFunc(key, want, bytes.Equal) {
			t.Errorf("wrong association between key and time: want %v, got %v", want, key)
		}
		if want := ("1" + string(byte(time))); id != want {
			t.Errorf("wrong association between id and time: want %v, got %v", want, id)
		}
		if want := (userhash.Hash{2, byte(time)}); user != want {
			t.Errorf("wrong association between user and time: want %v, got %v", want, user)
		}
	}
	// Do it again to verify we overwrite.
	for i := range len(p.id) {
		go func() {
			p.record("5"+string(byte(i)), userhash.Hash{6, byte(i)}, int64(i), [][]byte{{8, byte(i)}, {9}})
			ch <- struct{}{}
		}()
	}
	for range len(p.id) {
		<-ch
	}
	if p.k != 0 {
		t.Errorf("wrong final index: %d should be zero", p.k)
	}
	for k := range len(p.id) {
		key, id, user, time := p.key[k], p.id[k], p.user[k], p.time[k]
		if want := [][]byte{{8, byte(time)}, {9}}; !slices.EqualFunc(key, want, bytes.Equal) {
			t.Errorf("wrong association between key and time: want %v, got %v", want, key)
		}
		if want := ("5" + string(byte(time))); id != want {
			t.Errorf("wrong association between id and time: want %v, got %v", want, id)
		}
		if want := (userhash.Hash{6, byte(time)}); user != want {
			t.Errorf("wrong association between user and time: want %v, got %v", want, user)
		}
	}
}

func TestPastFindID(t *testing.T) {
	uu := "1"
	p := past{
		k:    127,
		key:  [256][][]byte{255: {[]byte("bocchi")}},
		id:   [256]string{255: uu},
		user: [256]userhash.Hash{255: {2}},
		time: [256]int64{255: 3},
	}
	if got, want := p.findID(uu), [][]byte{[]byte("bocchi")}; !slices.EqualFunc(got, want, bytes.Equal) {
		t.Errorf("wrong key: want %q, got %q", want, got)
	}
	if got := p.findID("fake"); got != nil {
		t.Errorf("non-nil key %q finding fake uuid", got)
	}
}

func TestPastFindDuring(t *testing.T) {
	p := past{
		k: 127,
		key: [256][][]byte{
			0: {[]byte("bocchi")},
			1: {[]byte("ryou")},
			2: {[]byte("nijika")},
			3: {[]byte("kita")},
		},
		id: [256]string{
			0: "2",
			1: "3",
			2: "4",
			3: "1",
		},
		user: [256]userhash.Hash{
			0: {3},
			1: {4},
			2: {5},
			3: {2},
		},
		time: [256]int64{
			0: 4,
			1: 5,
			2: 6,
			3: 3,
		},
	}
	want := [][]byte{
		[]byte("bocchi"),
		[]byte("ryou"),
	}
	got := p.findDuring(4, 5)
	if !slices.EqualFunc(got, want, bytes.Equal) {
		t.Errorf("wrong result: want %q, got %q", want, got)
	}
}

func TestPastFindUser(t *testing.T) {
	p := past{
		k: 127,
		key: [256][][]byte{
			127: {[]byte("bocchi")},
			192: {[]byte("ryou")},
			200: {[]byte("nijika")},
			255: {[]byte("kita")},
		},
		id: [256]string{
			127: "2",
			192: "3",
			200: "4",
			255: "1",
		},
		user: [256]userhash.Hash{
			127: {8},
			192: {8},
			200: {5},
			255: {5},
		},
		time: [256]int64{
			127: 4,
			192: 5,
			200: 6,
			255: 3,
		},
	}
	want := [][]byte{
		[]byte("bocchi"),
		[]byte("ryou"),
	}
	got := p.findUser(userhash.Hash{8})
	if !slices.EqualFunc(got, want, bytes.Equal) {
		t.Errorf("wrong result: want %q, got %q", want, got)
	}
}

func BenchmarkPastRecord(b *testing.B) {
	var p past
	uu := "1"
	user := userhash.Hash{2}
	b.ReportAllocs()
	for i := range b.N {
		p.record(uu, user, int64(i), [][]byte{{byte(i)}})
	}
}

func BenchmarkPastFindID(b *testing.B) {
	var p past
	for i := range len(p.id) {
		p.record(string(byte(i)), userhash.Hash{byte(i)}, int64(i), [][]byte{{byte(i)}})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		use(p.findID(string(byte(i))))
	}
}

func BenchmarkPastFindDuring(b *testing.B) {
	var p past
	for i := range len(p.id) {
		p.record(string(byte(i)), userhash.Hash{byte(i)}, int64(i), [][]byte{{byte(i)}})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		use(p.findDuring(int64(i), int64(i)))
	}
}

func BenchmarkPastFindUser(b *testing.B) {
	var p past
	for i := range len(p.id) {
		p.record(string(byte(i)), userhash.Hash{byte(i)}, int64(i), [][]byte{{byte(i)}})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		use(p.findUser(userhash.Hash{byte(i)}))
	}
}

//go:noinline
func use(x [][]byte) {}

func TestForgetMessage(t *testing.T) {
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
			uu:   "1",
			want: map[string]string{},
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
			uu:   "1",
			want: map[string]string{},
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
				err := br.Learn(ctx, msg.tag, msg.id, msg.user, msg.time, msg.tups)
				if err != nil {
					t.Errorf("failed to learn: %v", err)
				}
			}
			if err := br.ForgetMessage(ctx, "kessoku", c.uu); err != nil {
				t.Errorf("couldn't forget: %v", err)
			}
			dbcheck(t, db, c.want)
		})
	}
}

func TestForgetDuring(t *testing.T) {
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
		a, b int64
		want map[string]string
	}{
		{
			name: "single",
			msgs: []message{
				{
					id:   "1",
					user: userhash.Hash{2},
					tag:  "kessoku",
					time: time.Unix(1, 0),
					tups: []brain.Tuple{
						{
							Prefix: []string{"ryou", "bocchi"},
							Suffix: "kita",
						},
					},
				},
			},
			a:    0,
			b:    2,
			want: map[string]string{},
		},
		{
			name: "several",
			msgs: []message{
				{
					id:   "1",
					user: userhash.Hash{2},
					tag:  "kessoku",
					time: time.Unix(1, 0),
					tups: []brain.Tuple{
						{
							Prefix: []string{"ryou", "bocchi"},
							Suffix: "kita",
						},
					},
				},
				{
					id:   "2",
					user: userhash.Hash{2},
					tag:  "kessoku",
					time: time.Unix(1, 0),
					tups: []brain.Tuple{
						{
							Prefix: []string{"ryou", "bocchi"},
							Suffix: "kita",
						},
					},
				},
			},
			a:    0,
			b:    2,
			want: map[string]string{},
		},
		{
			name: "none",
			msgs: []message{
				{
					id:   "1",
					user: userhash.Hash{2},
					tag:  "kessoku",
					time: time.Unix(5, 0),
					tups: []brain.Tuple{
						{
							Prefix: []string{"ryou", "bocchi"},
							Suffix: "kita",
						},
					},
				},
			},
			a: 0,
			b: 2,
			want: map[string]string{
				mkey("kessoku", "ryou\xffbocchi\xff\xff", "1"): "kita",
			},
		},
		{
			name: "tagged",
			msgs: []message{
				{
					id:   "1",
					user: userhash.Hash{2},
					tag:  "sickhack",
					time: time.Unix(1, 0),
					tups: []brain.Tuple{
						{
							Prefix: []string{"ryou", "bocchi"},
							Suffix: "kita",
						},
					},
				},
			},
			a: 0,
			b: 2,
			want: map[string]string{
				mkey("sickhack", "ryou\xffbocchi\xff\xff", "1"): "kita",
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
				err := br.Learn(ctx, msg.tag, msg.id, msg.user, msg.time, msg.tups)
				if err != nil {
					t.Errorf("failed to learn: %v", err)
				}
			}
			since := time.Unix(c.a, 0)
			before := time.Unix(c.b, 0)
			if err := br.ForgetDuring(ctx, "kessoku", since, before); err != nil {
				t.Errorf("failed to forget between %v and %v: %v", since, before, err)
			}
			dbcheck(t, db, c.want)
		})
	}
}

func TestForgetUserSince(t *testing.T) {
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
		user userhash.Hash
		want map[string]string
	}{
		{
			name: "match",
			msgs: []message{
				{
					id:   "1",
					user: userhash.Hash{2},
					tag:  "kessoku",
					time: time.Unix(1, 0),
					tups: []brain.Tuple{
						{
							Prefix: []string{"ryou", "bocchi"},
							Suffix: "kita",
						},
					},
				},
			},
			user: userhash.Hash{2},
			want: map[string]string{},
		},
		{
			name: "different",
			msgs: []message{
				{
					id:   "1",
					user: userhash.Hash{2},
					tag:  "kessoku",
					time: time.Unix(1, 0),
					tups: []brain.Tuple{
						{
							Prefix: []string{"ryou", "bocchi"},
							Suffix: "kita",
						},
					},
				},
			},
			user: userhash.Hash{1},
			want: map[string]string{
				mkey("kessoku", "ryou\xffbocchi\xff\xff", "1"): "kita",
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
				err := br.Learn(ctx, msg.tag, msg.id, msg.user, msg.time, msg.tups)
				if err != nil {
					t.Errorf("failed to learn: %v", err)
				}
			}
			if err := br.ForgetUser(ctx, &c.user); err != nil {
				t.Errorf("failed to forget from user %02x: %v", c.user, err)
			}
			dbcheck(t, db, c.want)
		})
	}
}
