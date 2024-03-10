package kvbrain

import (
	"bytes"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

type past struct {
	mu sync.Mutex

	k    uint8
	key  [256][][]byte
	id   [256]uuid.UUID
	user [256]userhash.Hash
	time [256]int64 // unix nano
}

// record associates a message with a knowledge key.
func (p *past) record(id uuid.UUID, user userhash.Hash, nanotime int64, keys [][]byte) {
	p.mu.Lock()
	p.key[p.k] = slices.Grow(p.key[p.k][:0], len(keys))[:len(keys)]
	for i, key := range keys {
		p.key[p.k][i] = append(p.key[p.k][i][:0], key...)
	}
	p.id[p.k] = id
	p.user[p.k] = user
	p.time[p.k] = nanotime
	p.k++
	p.mu.Unlock()
}

// findID finds all keys corresponding to the given UUID.
func (p *past) findID(msg uuid.UUID) [][]byte {
	r := make([][]byte, 0, 64)
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.id {
		if v == msg {
			keys := p.key[k]
			r = slices.Grow(r, len(keys))
			for _, v := range keys {
				r = append(r, bytes.Clone(v))
			}
			return r
		}
	}
	return nil
}

// findDuring finds all knowledge keys of messages recorded with timestamps in
// the given time span.
func (p *past) findDuring(since, before int64) [][]byte {
	r := make([][]byte, 0, 64)
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.time {
		if since <= v && v <= before {
			keys := p.key[k]
			r = slices.Grow(r, len(keys))
			for _, v := range keys {
				r = append(r, bytes.Clone(v))
			}
		}
	}
	return r
}

// findUser finds all knowledge keys of messages recorded from a given user
// since a timestamp.
func (p *past) findUser(user userhash.Hash, since int64) [][]byte {
	r := make([][]byte, 0, 64)
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.time {
		if since <= v && p.user[k] == user {
			keys := p.key[k]
			r = slices.Grow(r, len(keys))
			for _, v := range keys {
				r = append(r, bytes.Clone(v))
			}
		}
	}
	return r
}

// Forget removes a set of recorded tuples. The tuples provided are as for
// Learn. If a tuple has been recorded multiple times, only the first
// should be deleted. If a tuple has not been recorded, it should be
// ignored.
func (br *Brain) Forget(ctx context.Context, tag string, tuples []brain.Tuple) error {
	panic("not implemented") // TODO: Implement
}

// ForgetMessage forgets everything learned from a single given message.
// If nothing has been learned from the message, it should be ignored.
func (br *Brain) ForgetMessage(ctx context.Context, tag string, msg uuid.UUID) error {
	panic("not implemented") // TODO: Implement
}

// ForgetDuring forgets all messages learned in the given time span.
func (br *Brain) ForgetDuring(ctx context.Context, tag string, since, before time.Time) error {
	panic("not implemented") // TODO: Implement
}

// ForgetUserSince forgets all messages learned from a user since a given
// time.
func (br *Brain) ForgetUserSince(ctx context.Context, user *userhash.Hash, since time.Time) error {
	panic("not implemented") // TODO: Implement
}
