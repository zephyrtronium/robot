package kvbrain

import (
	"bytes"
	"sync"

	"github.com/google/uuid"

	"github.com/zephyrtronium/robot/userhash"
)

type past struct {
	mu sync.Mutex

	k    uint8
	key  [256][]byte
	id   [256]uuid.UUID
	user [256]userhash.Hash
	time [256]int64 // unix nano
}

// record associates a message with a knowledge key.
func (p *past) record(id uuid.UUID, user userhash.Hash, nanotime int64, key []byte) {
	p.mu.Lock()
	p.key[p.k] = append(p.key[p.k][:0], key...)
	p.id[p.k] = id
	p.user[p.k] = user
	p.time[p.k] = nanotime
	p.k++
	p.mu.Unlock()
}

// findID finds the message with the given ID and appends its key to b.
// If the ID is not found in the past, the result is always nil.
func (p *past) findID(b []byte, msg uuid.UUID) []byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.id {
		if v == msg {
			return append(b, p.key[k]...)
		}
	}
	return nil
}

// findDuring finds all knowledge keys of messages recorded with timestamps in
// the given time span.
func (p *past) findDuring(since, before int64) [][]byte {
	r := make([][]byte, 0, 256)
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.time {
		if since <= v && v <= before {
			r = append(r, bytes.Clone(p.key[k]))
		}
	}
	return r
}

// findUser finds all knowledge keys of messages recorded from a given user
// since a timestamp.
func (p *past) findUser(user userhash.Hash, since int64) [][]byte {
	r := make([][]byte, 0, 256)
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.time {
		if since <= v && p.user[k] == user {
			r = append(r, bytes.Clone(p.key[k]))
		}
	}
	return r
}
