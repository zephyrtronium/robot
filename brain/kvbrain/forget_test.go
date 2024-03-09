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
			p.record(uuid.UUID{1, byte(i)}, userhash.Hash{2, byte(i)}, int64(i), []byte{4, byte(i)})
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
		if want := []byte{4, byte(time)}; !bytes.Equal(key, want) {
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
			p.record(uuid.UUID{5, byte(i)}, userhash.Hash{6, byte(i)}, int64(i), []byte{8, byte(i)})
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
		if want := []byte{8, byte(time)}; !bytes.Equal(key, want) {
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
		key:  [256][]byte{255: []byte("bocchi")},
		id:   [256]uuid.UUID{255: uu},
		user: [256]userhash.Hash{255: {2}},
		time: [256]int64{255: 3},
	}
	if got := p.findID(nil, uu); string(got) != "bocchi" {
		t.Errorf("wrong key: want %q, got %q", "bocchi", got)
	}
	if got := p.findID([]byte{}, uuid.Max); got != nil {
		t.Errorf("non-nil key %q finding fake uuid", got)
	}
}

func TestPastFindDuring(t *testing.T) {
	p := past{
		k: 127,
		key: [256][]byte{
			127: []byte("bocchi"),
			192: []byte("ryou"),
			200: []byte("nijika"),
			255: []byte("kita"),
		},
		id: [256]uuid.UUID{
			127: {2},
			192: {3},
			200: {4},
			255: {1},
		},
		user: [256]userhash.Hash{
			127: {3},
			192: {4},
			200: {5},
			255: {2},
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
	got := p.findDuring(4, 5)
	if !slices.EqualFunc(got, want, bytes.Equal) {
		t.Errorf("wrong result: want %q, got %q", want, got)
	}
}

func TestPastFindUser(t *testing.T) {
	p := past{
		k: 127,
		key: [256][]byte{
			127: []byte("bocchi"),
			192: []byte("ryou"),
			200: []byte("nijika"),
			255: []byte("kita"),
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
