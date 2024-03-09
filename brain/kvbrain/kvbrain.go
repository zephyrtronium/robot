package kvbrain

import (
	"context"
	"hash/fnv"
	"io"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

/*
Message key structure:
Tag × Tuples × UUID
- Tag is a 16 byte string padded with \x00.
- Tuple terms are separated by \xff sentinels. Terms are recorded in reverse order.
- The final tuple term is the empty string, so the tuple portion ends with \xff\xff.
- UUID is the raw uuid.

As with the SQL approach, we record every prefix with its suffix, including the
final empty prefix.

Operations:
- Find a start tuple: Search for a prefix of tag × \xff.
- Find a continuation:
	+ With full context, just search for it, again in reverse order.
	+ When we reduce context, record by how much and only search for that much.
	+ In both cases, and with start tuple, check message UUID and tags we
		select against the deletions db.
- Learn: Construct the key according to above. The suffix is the entire value.
	Record a mapping of tag, UUID, timestamp, and userhash to keys.
- Forget tuples: thinking…
- ForgetMessage, ForgetDuring, ForgetUserSince: Look up the actual keys to
	delete in the recording taken during learning.
*/

type Brain struct {
	knowledge *badger.DB
}

var _ brain.Learner = (*Brain)(nil)

func New(knowledge *badger.DB) *Brain {
	return &Brain{
		knowledge: knowledge,
	}
}

// Order returns the number of elements in the prefix of a chain. It is
// called once at the beginning of learning. The returned value must always
// be at least 1.
func (br *Brain) Order() int {
	// TOOD(zeph): this can go away one day
	return 250
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

func hashTag(tag string) uint64 {
	h := fnv.New64a()
	io.WriteString(h, tag)
	return h.Sum64()
}
