package kvbrain

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/zephyrtronium/robot/userhash"
)

type past struct {
	mu sync.Mutex

	k    uint8
	key  [256][][]byte
	id   [256]string
	user [256]userhash.Hash
	time [256]int64 // unix nano
}

// record associates a message with a knowledge key.
func (p *past) record(id string, user userhash.Hash, nanotime int64, keys [][]byte) {
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
func (p *past) findID(id string) [][]byte {
	r := make([][]byte, 0, 64)
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.id {
		if v == id {
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

// Forget forgets everything learned from a single given message.
// If nothing has been learned from the message, it should be ignored.
func (br *Brain) Forget(ctx context.Context, tag, id string) error {
	past, _ := br.past.Load(tag)
	if past == nil {
		return nil
	}
	keys := past.findID(id)
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
		return fmt.Errorf("couldn't commit deleting message %v: %w", id, err)
	}
	return nil
}
