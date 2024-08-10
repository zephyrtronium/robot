package kvbrain

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/dgraph-io/badger/v4"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/prepend"
	"github.com/zephyrtronium/robot/tpool"
)

var prependerPool tpool.Pool[*prepend.List[string]]

// Speak generates a full message and appends it to w.
// The prompt is in reverse order and has entropy reduction applied.
func (br *Brain) Speak(ctx context.Context, tag string, prompt []string, w []byte) ([]byte, error) {
	search := prependerPool.Get().Set(prompt...)
	defer func() { prependerPool.Put(search) }()

	tb := hashTag(make([]byte, 0, tagHashLen), tag)
	b := make([]byte, 0, 128)
	opts := badger.DefaultIteratorOptions
	// We don't actually need to iterate over values, only the single value
	// that we decide to use per suffix. So, we can disable value prefetch.
	opts.PrefetchValues = false
	opts.Prefix = hashTag(nil, tag)
	for range 1024 {
		var err error
		var l int
		b = append(b[:0], tb...)
		b, l, err = br.next(b, search.Slice(), opts)
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			break
		}
		w = append(w, b...)
		search = search.Drop(search.Len() - l - 1).Prepend(brain.ReduceEntropy(string(b)))
	}
	return w, nil
}

// next finds a single token to continue a prompt.
// The returned values are, in order,
// b with its contents replaced with the new term,
// the number of terms of the prompt which matched to produce the new term,
// and any error.
// If the returned term is the empty string, generation should end.
func (br *Brain) next(b []byte, prompt []string, opts badger.IteratorOptions) ([]byte, int, error) {
	// These definitions are outside the loop to ensure we don't bias toward
	// smaller contexts.
	var (
		key    []byte
		skip   brain.Skip
		picked int
		n      uint64
	)
	b = appendPrefix(b, prompt)
	if len(prompt) == 0 {
		// If we have no prompt, then we want to make sure we select only
		// options that start a message.
		b = append(b, '\xff')
	}
	for {
		err := br.knowledge.View(func(txn *badger.Txn) error {
			it := txn.NewIterator(opts)
			defer it.Close()
			it.Seek(b)
			for it.ValidForPrefix(b) {
				if n == 0 {
					item := it.Item()
					// TODO(zeph): for #43, check deleted uuids so we never
					// pick a message that has been deleted
					key = item.KeyCopy(key[:0])
					n = skip.N(rand.Uint64(), rand.Uint64())
				}
				it.Next()
				n--
			}
			return nil
		})
		if err != nil {
			return nil, len(prompt), fmt.Errorf("couldn't read knowledge: %w", err)
		}
		if picked < 3 && len(prompt) > 1 {
			// We haven't seen enough options, and we have context we could
			// lose. Do so and try again from the beginning.
			prompt = prompt[:len(prompt)-1]
			b = appendPrefix(b[:tagHashLen], prompt)
			continue
		}
		if key == nil {
			// We never saw any options. Since we always select the first, this
			// means there were no options. Don't look for nothing in the DB.
			return b[:0], len(prompt), nil
		}
		err = br.knowledge.View(func(txn *badger.Txn) error {
			item, err := txn.Get(key)
			if err != nil {
				return fmt.Errorf("couldn't get item for key %q: %w", key, err)
			}
			b, err = item.ValueCopy(b[:0])
			if err != nil {
				return fmt.Errorf("couldn't get value for key %q: %w", key, err)
			}
			return nil
		})
		return b, len(prompt), err
	}
}
