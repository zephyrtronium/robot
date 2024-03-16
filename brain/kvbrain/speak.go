package kvbrain

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/dgraph-io/badger/v4"

	"github.com/zephyrtronium/robot/brain"
)

// New finds a prompt to begin a random message. When a message is
// generated with no prompt, the result from New is passed directly to
// Speak; it is the speaker's responsibility to ensure it meets
// requirements with regard to length and matchable content. Only data
// originally learned with the given tag should be used to generate a
// prompt.
func (br *Brain) New(ctx context.Context, tag string) ([]string, error) {
	return br.Speak(ctx, tag, nil)
}

// Speak generates a full message from the given prompt. The prompt is
// guaranteed to have length equal to the value returned from Order, unless
// it is a prompt returned from New. If the number of tokens in the prompt
// is smaller than Order, the difference is made up by prepending empty
// strings to the prompt. The speaker should use ReduceEntropy on all
// tokens, including those in the prompt, when generating a message.
// Empty strings at the start and end of the result will be trimmed. Only
// data originally learned with the given tag should be used to generate a
// message.
func (br *Brain) Speak(ctx context.Context, tag string, prompt []string) ([]string, error) {
	terms := make([]string, 0, len(prompt))
	for i, s := range prompt {
		if s == "" {
			continue
		}
		terms = append(terms, s)
		prompt[i] = brain.ReduceEntropy(s)
	}
	var b []byte
	opts := badger.DefaultIteratorOptions
	// We don't actually need to iterate over values, only the single value
	// that we decide to use per suffix. So, we can disable value prefetch.
	opts.PrefetchValues = false
	opts.Prefix = hashTag(nil, tag)
	for {
		var err error
		var s string
		b = hashTag(b[:0], tag)
		s, b, prompt, err = br.next(b, prompt, opts)
		if err != nil {
			return nil, err
		}
		if s == "" {
			return terms, nil
		}
		terms = append(terms, s)
		prompt = append(prompt, brain.ReduceEntropy(s))
	}
}

// next finds a single token to continue a prompt.
// The returned values are, in order, the new term, b with possibly appended
// memory, the suffix of prompt which matched to produce the new term, and
// any error. If the returned term is the empty string, generation should end.
func (br *Brain) next(b []byte, prompt []string, opts badger.IteratorOptions) (string, []byte, []string, error) {
	// These definitions are outside the loop to ensure we don't bias toward
	// smaller contexts.
	var (
		key    []byte
		m      uint64
		picked int
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
				// We generate a uniform variate per key, then choose the key
				// that gets the maximum variate.
				u := rand.Uint64()
				if m <= u {
					item := it.Item()
					// TODO(zeph): for #43, check deleted uuids so we never
					// pick a message that has been deleted
					key = item.KeyCopy(key[:0])
					m = u
					picked++
				}
				it.Next()
			}
			return nil
		})
		if err != nil {
			return "", b, prompt, fmt.Errorf("couldn't read knowledge: %w", err)
		}
		if picked < 3 && len(prompt) > 1 {
			// We haven't seen enough options, and we have context we could
			// lose. Do so and try again from the beginning.
			// TODO(zeph): we could save the start of the prompt so we don't
			// reallocate, and we could construct the next key to use by
			// trimming off the end of the current one
			prompt = prompt[1:]
			b = appendPrefix(b[:8], prompt)
			continue
		}
		if key == nil {
			// We never saw any options. Since we always select the first, this
			// means there were no options. Don't look for nothing in the DB.
			return "", b, prompt, nil
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
		if err != nil {
			return "", b, prompt, err
		}
		return string(b), b, prompt, nil
	}
}
