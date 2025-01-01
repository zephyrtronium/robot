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
		d := make([]byte, 0, 64)
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
				_, _, t := keyparts(key)
				*id = append(*id, t...)
				var err error
				*suf, err = item.ValueCopy(*suf)
				if err != nil {
					return fmt.Errorf("couldn't get item for key %q: %w", key, err)
				}
				return nil
			}
			for it.ValidForPrefix(b) {
				item = it.Item()
				key = item.KeyCopy(key[:0])
				// Check whether the message ID is deleted.
				tag, _, id := keyparts(key)
				d = append(d[:0], tag...)
				d = append(d, 0xfe, 0xfe)
				d = append(d, id...)
				switch _, err := txn.Get(d); err {
				case badger.ErrKeyNotFound: // do nothing
				case nil:
					// The fact that there is a value means the message was
					// forgotten. Don't yield it.
					it.Next()
					continue
				default:
					erf(fmt.Errorf("couldn't check for deleted message: %w", err))
				}
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

func keyparts(key []byte) (tag, content, id []byte) {
	if len(key) < tagHashLen+2 {
		return nil, nil, nil
	}
	tag = key[:tagHashLen]
	switch key[tagHashLen] {
	case 0xff:
		// Empty prefix sentinel. The rest is the ID.
		content = key[tagHashLen : tagHashLen+1]
		id = key[tagHashLen+1:]
	case 0xfe:
		// Deleted ID sentinel. Two bytes long, and the rest is the ID.
		content = key[tagHashLen : tagHashLen+2]
		id = key[tagHashLen+2:]
	default:
		// Non-empty prefix. Ends after \xff\xff.
		k := bytes.Index(key[tagHashLen:], []byte{0xff, 0xff})
		if k < 0 {
			panic("kvbrain: invalid key")
		}
		content = key[tagHashLen : tagHashLen+k+2]
		id = key[tagHashLen+k+2:]
	}
	return tag, content, id
}
