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
	type entry struct {
		key []byte
		val []byte
	}
	entries := make([]entry, len(tuples))
	var b bytes.Buffer
	p := make([]string, 0, len(tuples[0].Prefix))
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
		p = append(p[:0], t.Prefix[k:]...)
		slices.Reverse(p)
		for _, s := range p {
			b.WriteString(s)
			b.WriteByte('\xff')
		}
		b.WriteByte('\xff')
		// Write message ID.
		b.Write(meta.ID[:])
		entries[i] = entry{
			key: bytes.Clone(b.Bytes()),
			val: []byte(t.Suffix),
		}
	}
	// TODO(zeph): record mapping of metadata to key
	batch := br.knowledge.NewWriteBatch()
	defer batch.Cancel()
	for _, e := range entries {
		err := batch.Set(e.key, e.val)
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
