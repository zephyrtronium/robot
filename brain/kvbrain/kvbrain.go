package kvbrain

import (
	"hash/fnv"
	"io"

	"github.com/dgraph-io/badger/v4"
	"gopkg.in/typ.v4/sync2"

	"github.com/zephyrtronium/robot/brain"
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
	past      sync2.Map[string, *past]
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

// hashTag appends the hash of a tag to b to serve as the start of a knowledge key.
func hashTag(b []byte, tag string) []byte {
	h := fnv.New64a()
	io.WriteString(h, tag)
	return h.Sum(b)
}
