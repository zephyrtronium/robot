// Package userhash provides an obfuscation layer for users in locations.
//
// A userhash allows messages from a given user to be correlated within a
// channel, but not easily between channels. Specifically, a userhash is based
// on HMAC with the user ID, channel name, and a value derived from the time as
// the message content.
//
// The key used to generate hashes must be preserved across program instances.
//
// Userhashes are not intended to guarantee privacy.
package userhash

import (
	"crypto/hmac"
	"encoding/binary"
	"errors"
	"hash"
	"time"

	"golang.org/x/crypto/sha3"
)

// Size is the size of a userhash in bytes.
const Size = 28

// TimeQuantum is the duration for which hashing a user and location gives the
// same result.
const TimeQuantum = 15 * time.Minute

var (
	// ErrShortHash is an error returned when scanning a userhash that is too
	// short.
	ErrShortHash = errors.New("short userhash")
	// ErrHashType is an error returned when scanning a userhash from a type
	// that cannot be handled.
	ErrHashType = errors.New("bad type for userhash")
)

// Hash is an obfuscated hash identifying a user in a location.
type Hash [Size]byte

// Scan implements sql.Scanner.
func (h *Hash) Scan(src any) error {
	switch src := src.(type) {
	case []byte:
		n := copy(h[:], src)
		if n != Size {
			return ErrShortHash
		}
	default:
		return ErrHashType
	}
	return nil
}

// A Hasher creates Hash values.
type Hasher struct {
	// mac is the HMAC hasher.
	mac hash.Hash
}

// New creates a Hasher.
func New(prk []byte) Hasher {
	return Hasher{
		mac: hmac.New(sha3.New224, prk),
	}
}

// Hash computes a userhash and writes it into dst.
func (h Hasher) Hash(dst *Hash, uid, where string, when time.Time) *Hash {
	h.mac.Reset()
	t := when.UnixNano() / TimeQuantum.Nanoseconds()
	b := make([]byte, 8, 8+len(uid)+1+len(where))
	binary.LittleEndian.PutUint64(b, uint64(t))
	b = append(b, uid...)
	b = append(b, 0xaa)
	b = append(b, where...)
	h.mac.Write(b)
	return (*Hash)(h.mac.Sum(dst[:0]))
}
