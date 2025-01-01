package kvbrain

import (
	"context"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

// Forget forgets everything learned from a single given message.
// If nothing has been learned from the message, it should be ignored.
func (br *Brain) Forget(ctx context.Context, tag, id string) error {
	err := br.knowledge.Update(func(txn *badger.Txn) error {
		k := make([]byte, 0, tagHashLen+2+len(id))
		k = hashTag(k, tag)
		k = append(k, 0xfe, 0xfe)
		k = append(k, id...)
		return txn.Set(k, []byte{})
	})
	if err != nil {
		return fmt.Errorf("couldn't forget: %w", err)
	}
	return nil
}
