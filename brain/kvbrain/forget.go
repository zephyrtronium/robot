package kvbrain

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
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
	// Sort tuples so that we always seek forward.
	slices.SortFunc(tuples, func(a, b brain.Tuple) int {
		p := slices.Compare(a.Prefix, b.Prefix)
		if p == 0 {
			p = strings.Compare(a.Suffix, b.Suffix)
		}
		return p
	})
	err := br.knowledge.Update(func(txn *badger.Txn) error {
		th := hashTag(tag)
		opts := badger.DefaultIteratorOptions
		opts.Prefix = binary.LittleEndian.AppendUint64(nil, th)
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		var b []byte
		for _, t := range tuples {
			b = keystart(b[:0], tag, t.Prefix)
			it.Seek(b)
			for it.ValidForPrefix(b) {
				v := it.Item()
				it.Next()
				if v.IsDeletedOrExpired() {
					continue
				}
				var err error
				b, err = v.ValueCopy(b[:0])
				if err != nil {
					// TODO(zeph): collect and continue
					return err
				}
				if string(b) != t.Suffix {
					continue
				}
				if err := txn.Delete(v.KeyCopy(nil)); err != nil {
					// TODO(zeph): collect and continue
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("couldn't forget: %w", err)
	}
	return nil
}

// ForgetMessage forgets everything learned from a single given message.
// If nothing has been learned from the message, it should be ignored.
func (br *Brain) ForgetMessage(ctx context.Context, tag string, msg uuid.UUID) error {
	past, _ := br.past.Load(tag)
	if past == nil {
		return nil
	}
	keys := past.findID(msg)
	batch := br.knowledge.NewWriteBatch()
	defer batch.Cancel()
	for _, key := range keys {
		err := batch.Delete(key)
		if err != nil {
			return err
		}
	}
	err := batch.Flush()
	if err != nil {
		return fmt.Errorf("couldn't commit deleting message %v: %w", msg, err)
	}
	return nil
}

// ForgetDuring forgets all messages learned in the given time span.
func (br *Brain) ForgetDuring(ctx context.Context, tag string, since, before time.Time) error {
	past, _ := br.past.Load(tag)
	if past == nil {
		return nil
	}
	keys := past.findDuring(since.UnixNano(), before.UnixNano())
	batch := br.knowledge.NewWriteBatch()
	defer batch.Cancel()
	for _, key := range keys {
		err := batch.Delete(key)
		if err != nil {
			return err
		}
	}
	err := batch.Flush()
	if err != nil {
		return fmt.Errorf("couldn't commit deleting between times %v and %v: %w", since, before, err)
	}
	return nil
}

// ForgetUserSince forgets all messages learned from a user since a given
// time.
func (br *Brain) ForgetUserSince(ctx context.Context, user *userhash.Hash, since time.Time) error {
	var rangeErr error
	u := *user
	br.past.Range(func(tag string, past *past) bool {
		keys := past.findUser(u, since.UnixNano())
		if len(keys) == 0 {
			return true
		}
		batch := br.knowledge.NewWriteBatch()
		defer batch.Cancel()
		for _, key := range keys {
			err := batch.Delete(key)
			if err != nil {
				rangeErr = err
				return false
			}
		}
		err := batch.Flush()
		if err != nil {
			rangeErr = fmt.Errorf("couldn't commit deleting messages from user since %v: %w", since, err)
			return false
		}
		return false
	})
	return rangeErr
}
