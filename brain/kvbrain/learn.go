package kvbrain

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"

	"github.com/zephyrtronium/robot/brain"
)

// Learn records a set of tuples. Each tuple prefix has length equal to the
// result of Order. The tuples begin with empty strings in the prefix to
// denote the start of the message and end with one empty suffix to denote
// the end; all other tokens are non-empty. Each tuple's prefix has entropy
// reduction transformations applied.
func (br *Brain) Learn(ctx context.Context, meta *brain.MessageMeta, tuples []brain.Tuple) error {
	if len(tuples) == 0 {
		return errors.New("no tuples to learn")
	}
	// Construct the keys and values we will use.
	// There are probably things we could do to control allocations since we're
	// using many overlapping tuples for keys, but it's tremendously easier to
	// just fill up a buffer for each.
	keys := make([][]byte, len(tuples))
	vals := make([][]byte, len(tuples)) // TODO(zeph): could do one call to make
	var b bytes.Buffer
	pre := make([]string, 0, len(tuples[0].Prefix))
	for i, t := range tuples {
		b.Reset()
		// Write the tag.
		u := make([]byte, 8)
		binary.LittleEndian.PutUint64(u, hashTag(meta.Tag))
		b.Write(u)
		// Write prefixes.
		k := slices.IndexFunc(t.Prefix, func(s string) bool { return s != "" })
		if k < 0 {
			// First prefix of the message. We want to write only the separator.
			k = len(t.Prefix)
		}
		pre = append(pre[:0], t.Prefix[k:]...)
		slices.Reverse(pre)
		for _, s := range pre {
			b.WriteString(s)
			b.WriteByte('\xff')
		}
		b.WriteByte('\xff')
		// Write message ID.
		b.Write(meta.ID[:])
		keys[i] = bytes.Clone(b.Bytes())
		vals[i] = []byte(t.Suffix)
	}

	p, _ := br.past.Load(meta.Tag)
	if p == nil {
		// We might race with others also creating this past. Ensure we don't
		// overwrite if that happens.
		p, _ = br.past.LoadOrStore(meta.Tag, new(past))
	}
	p.record(meta.ID, meta.User, meta.Time.UnixNano(), keys)

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
