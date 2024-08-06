package kvbrain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

// Learn records a set of tuples. Each tuple prefix has length equal to the
// result of Order. The tuples begin with empty strings in the prefix to
// denote the start of the message and end with one empty suffix to denote
// the end; all other tokens are non-empty. Each tuple's prefix has entropy
// reduction transformations applied.
func (br *Brain) Learn(ctx context.Context, tag string, user userhash.Hash, id uuid.UUID, t time.Time, tuples []brain.Tuple) error {
	if len(tuples) == 0 {
		return errors.New("no tuples to learn")
	}
	// Construct the keys and values we will use.
	// There are probably things we could do to control allocations since we're
	// using many overlapping tuples for keys, but it's tremendously easier to
	// just fill up a buffer for each.
	keys := make([][]byte, len(tuples))
	vals := make([][]byte, len(tuples)) // TODO(zeph): could do one call to make
	var b []byte
	for i, t := range tuples {
		b = hashTag(b[:0], tag)
		b = append(appendPrefix(b, t.Prefix), '\xff')
		// Write message ID.
		b = append(b, id[:]...)
		keys[i] = bytes.Clone(b)
		vals[i] = []byte(t.Suffix)
	}

	p, _ := br.past.Load(tag)
	if p == nil {
		// We might race with others also creating this past. Ensure we don't
		// overwrite if that happens.
		p, _ = br.past.LoadOrStore(tag, new(past))
	}
	p.record(id, user, t.UnixNano(), keys)

	batch := br.knowledge.NewWriteBatch()
	defer batch.Cancel()
	for i, key := range keys {
		err := batch.Set(key, vals[i])
		if err != nil {
			return err
		}
	}
	err := batch.Flush()
	if err != nil {
		return fmt.Errorf("couldn't commit learned knowledge: %w", err)
	}
	return nil
}

// appendPrefix appends the prefix components for a knowledge key to b,
// not including the sentinel marking the end of the prefix. To serve as a
// knowledge key, b should already contain the hashed tag. The caller should
// append a final \xff to terminate the prefix before appending the message ID
// to form a complete key.
func appendPrefix(b []byte, prefix []string) []byte {
	for _, w := range prefix {
		b = append(b, w...)
		b = append(b, '\xff')
	}
	return b
}
