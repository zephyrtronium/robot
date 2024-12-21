package kvbrain

import (
	"bytes"
	"context"
	"fmt"
	"iter"
	"math/rand/v2"

	"github.com/dgraph-io/badger/v4"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/deque"
	"github.com/zephyrtronium/robot/tpool"
)

var prependerPool tpool.Pool[deque.Deque[string]]

func (br *Brain) Think(ctx context.Context, tag string, prompt []string) iter.Seq[func(id *[]byte, suf *[]byte) error] {
	return func(yield func(func(id *[]byte, suf *[]byte) error) bool) {
		erf := func(err error) { yield(func(id, suf *[]byte) error { return err }) }
		opts := badger.DefaultIteratorOptions
		opts.Prefix = hashTag(make([]byte, 0, tagHashLen), tag)
		// We don't actually need to iterate over values, only the single value
		// that we decide to use per suffix. So, we can disable value prefetch.
		opts.PrefetchValues = false
		b := make([]byte, 0, 128)
		b = append(b, opts.Prefix...)
		b = appendPrefix(b, prompt)
		if len(prompt) == 0 {
			// If we have no prompt, then we want to make sure we select only
			// options that start a message.
			// TODO(zeph): a better way to do this would be to use \x00 as the
			// terminator so that we can append this unconditionally and then
			// we always seek to the start of the iteration.
			b = append(b, '\xff')
		}
		var key []byte
		var item *badger.Item
		err := br.knowledge.View(func(txn *badger.Txn) error {
			it := txn.NewIterator(opts)
			defer it.Close()
			it.Seek(b)
			// NOTE(zeph): We're doing some cursed things to limit allocations.
			// We yield this same closure on every iteration of the loop, but
			// it uses variables key and item that aren't actually set until
			// after its definition, because our half of the loop needs to use
			// them too.
			f := func(id, suf *[]byte) error {
				// The id is everything after the first byte following the hash
				// for empty prefixes, and everything after the first \xff\xff
				// otherwise.
				k := key[tagHashLen+1:]
				if len(prompt) > 0 {
					_, k, _ = bytes.Cut(k, []byte{0xff, 0xff})
				}
				*id = k
				var err error
				*suf, err = item.ValueCopy(*suf)
				if err != nil {
					return fmt.Errorf("couldn't get item for key %q: %w", key, err)
				}
				return nil
			}
			for it.ValidForPrefix(b) {
				item = it.Item()
				// TODO(zeph): for #43, check deleted uuids so we never pick
				// a message that has been deleted
				key = item.KeyCopy(key[:0])
				if !yield(f) {
					break
				}
				it.Next()
			}
			return nil
		})
		if err != nil {
			erf(fmt.Errorf("couldn't read knowledge: %w", err))
		}
	}
}

// Speak generates a full message and appends it to w.
// The prompt is in reverse order and has entropy reduction applied.
func (br *Brain) Speak(ctx context.Context, tag string, prompt []string, w *brain.Builder) error {
	search := prependerPool.Get().Prepend(prompt...)
	defer func() { prependerPool.Put(search.Reset()) }()

	tb := hashTag(make([]byte, 0, tagHashLen), tag)
	b := make([]byte, 0, 128)
	var id string
	opts := badger.DefaultIteratorOptions
	// We don't actually need to iterate over values, only the single value
	// that we decide to use per suffix. So, we can disable value prefetch.
	opts.PrefetchValues = false
	opts.Prefix = hashTag(nil, tag)
	for range 1024 {
		var err error
		var l int
		b = append(b[:0], tb...)
		b, id, l, err = br.next(b, search.Slice(), opts)
		if err != nil {
			return err
		}
		if len(b) == 0 {
			break
		}
		w.Append(id, b)
		search = search.DropEnd(search.Len() - l - 1).Prepend(brain.ReduceEntropy(string(b)))
	}
	return nil
}

// next finds a single token to continue a prompt.
// The returned values are, in order,
// b with its contents replaced with the new term,
// the ID of the message used for the term,
// the number of terms of the prompt which matched to produce the new term,
// and any error.
// If the returned term is the empty string, generation should end.
func (br *Brain) next(b []byte, prompt []string, opts badger.IteratorOptions) ([]byte, string, int, error) {
	// These definitions are outside the loop to ensure we don't bias toward
	// smaller contexts.
	var (
		key  []byte
		skip brain.Skip
		n    uint64
	)
	b = appendPrefix(b, prompt)
	if len(prompt) == 0 {
		// If we have no prompt, then we want to make sure we select only
		// options that start a message.
		b = append(b, '\xff')
	}
	for {
		var seen uint64
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
				seen++
			}
			return nil
		})
		if err != nil {
			return nil, "", len(prompt), fmt.Errorf("couldn't read knowledge: %w", err)
		}
		// Try to lose context.
		// We want to do so when we have a long context and almost no options,
		// or at random with even a short context.
		// Note that in the latter case we use a 1/2 chance; it seems high, but
		// n.b. the caller will recover the last token that we discard.
		if len(prompt) > 4 && seen <= 2 || len(prompt) > 2 && rand.Uint32()&1 == 0 {
			// We haven't seen enough options, and we have context we could
			// lose. Do so and try again from the beginning.
			prompt = prompt[:len(prompt)-1]
			b = appendPrefix(b[:tagHashLen], prompt)
			continue
		}
		if key == nil {
			// We never saw any options. Since we always select the first, this
			// means there were no options. Don't look for nothing in the DB.
			return b[:0], "", len(prompt), nil
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
		// The id is everything after the first byte following the hash for
		// empty prefixes, and everything after the first \xff\xff otherwise.
		id := key[tagHashLen+1:]
		if len(prompt) > 0 {
			_, id, _ = bytes.Cut(key, []byte{0xff, 0xff})
		}
		return b, string(id), len(prompt), err
	}
}
