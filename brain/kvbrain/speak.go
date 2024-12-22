package kvbrain

import (
	"bytes"
	"context"
	"fmt"
	"iter"

	"github.com/dgraph-io/badger/v4"
)

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
