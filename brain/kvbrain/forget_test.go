package kvbrain

import (
	"bytes"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/zephyrtronium/robot/userhash"
)

func TestPastRecord(t *testing.T) {
	var p past
	ch := make(chan struct{})
	for i := range len(p.id) {
		go func() {
			p.record(uuid.UUID{1, byte(i)}, userhash.Hash{2, byte(i)}, int64(i), [][]byte{{4, byte(i)}, {5}})
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
		if want := (uuid.UUID{1, byte(time)}); id != want {
			t.Errorf("wrong association between id and time: want %v, got %v", want, id)
		}
		if want := (userhash.Hash{2, byte(time)}); user != want {
			t.Errorf("wrong association between user and time: want %v, got %v", want, user)
		}
	}
	// Do it again to verify we overwrite.
	for i := range len(p.id) {
		go func() {
			p.record(uuid.UUID{5, byte(i)}, userhash.Hash{6, byte(i)}, int64(i), [][]byte{{8, byte(i)}, {9}})
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
		if want := (uuid.UUID{5, byte(time)}); id != want {
			t.Errorf("wrong association between id and time: want %v, got %v", want, id)
		}
		if want := (userhash.Hash{6, byte(time)}); user != want {
			t.Errorf("wrong association between user and time: want %v, got %v", want, user)
		}
	}
}

func TestPastFindID(t *testing.T) {
	uu := uuid.UUID{1}
	p := past{
		k:    127,
		key:  [256][][]byte{255: {[]byte("bocchi")}},
		id:   [256]uuid.UUID{255: uu},
		user: [256]userhash.Hash{255: {2}},
		time: [256]int64{255: 3},
	}
	if got, want := p.findID(uu), [][]byte{[]byte("bocchi")}; !slices.EqualFunc(got, want, bytes.Equal) {
		t.Errorf("wrong key: want %q, got %q", want, got)
	}
	if got := p.findID(uuid.Max); got != nil {
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
		id: [256]uuid.UUID{
			0: {2},
			1: {3},
			2: {4},
			3: {1},
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
		id: [256]uuid.UUID{
			127: {2},
			192: {3},
			200: {4},
			255: {1},
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
		[]byte("ryou"),
	}
	got := p.findUser(userhash.Hash{8}, 5)
	if !slices.EqualFunc(got, want, bytes.Equal) {
		t.Errorf("wrong result: want %q, got %q", want, got)
	}
}

func BenchmarkPastRecord(b *testing.B) {
	var p past
	uu := uuid.UUID{1}
	user := userhash.Hash{2}
	b.ReportAllocs()
	for i := range b.N {
		p.record(uu, user, int64(i), [][]byte{{byte(i)}})
	}
}

func BenchmarkPastFindID(b *testing.B) {
	var p past
	for i := range len(p.id) {
		p.record(uuid.UUID{byte(i)}, userhash.Hash{byte(i)}, int64(i), [][]byte{{byte(i)}})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		use(p.findID(uuid.UUID{byte(i)}))
	}
}

func BenchmarkPastFindDuring(b *testing.B) {
	var p past
	for i := range len(p.id) {
		p.record(uuid.UUID{byte(i)}, userhash.Hash{byte(i)}, int64(i), [][]byte{{byte(i)}})
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
		p.record(uuid.UUID{byte(i)}, userhash.Hash{byte(i)}, int64(i), [][]byte{{byte(i)}})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		use(p.findUser(userhash.Hash{byte(i)}, 0))
	}
}

//go:noinline
func use(x [][]byte) {}
