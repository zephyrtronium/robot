package auth

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/chacha20poly1305"
)

// Storage is a secure means to store OAuth2 credentials.
type Storage interface {
	// Load returns the current refresh token. If the result is the empty
	// string, the authorization code grant flow is triggered.
	Load(ctx context.Context) (string, error)
	// Store sets a new refresh token. If rt is the empty string, the storage
	// should be cleared.
	Store(ctx context.Context, rt string) error
}

// file is the interface used by a FileStorage.
type file interface {
	io.ReaderAt
	io.WriterAt
	Truncate(int64) error
}

// FileStorage is an encrypted file storage for OAuth2 credentials.
type FileStorage struct {
	f    file
	enc  cipher.AEAD
	rand io.Reader
}

// KeySize is the size of the key used to encrypt the token file.
const KeySize = chacha20poly1305.KeySize

const (
	nonceSize = chacha20poly1305.NonceSize
	totalOH   = chacha20poly1305.NonceSize + chacha20poly1305.Overhead
)

// NewFileAt creates a FileStorage at path p.
func NewFileAt(p string, key [KeySize]byte) (*FileStorage, error) {
	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	enc, err := chacha20poly1305.New(key[:])
	if err != nil {
		panic(err)
	}
	return &FileStorage{f: f, enc: enc, rand: rand.Reader}, nil
}

// Load decrypts the token value. If there is no token, the result is the empty
// string with a nil error.
func (f *FileStorage) Load(ctx context.Context) (string, error) {
	_, p, err := f.parts()
	return string(p), err
}

// Store sets a new token value. If the token file contains data that is not a
// valid token encrypted with the key passed to NewFileAt, Store returns an
// error.
func (f *FileStorage) Store(ctx context.Context, rt string) error {
	if rt == "" {
		// Clear the existing token.
		err := f.f.Truncate(0)
		if err != nil {
			return fmt.Errorf("couldn't clear token: %w", err)
		}
		return nil
	}
	b, _, err := f.parts()
	if err != nil {
		return err
	}
	if len(b) == 0 {
		// File is empty. We'll be initializing it.
		b = initialNonce(rt, f.rand)
	}
	v := binary.LittleEndian.Uint64(b)
	v++
	binary.LittleEndian.PutUint64(b, v)
	r := f.enc.Seal(b, b, []byte(rt), nil)
	if _, err := f.f.WriteAt(r, 0); err != nil {
		return fmt.Errorf("couldn't save refresh token: %w", err)
	}
	return nil
}

func (f *FileStorage) parts() (nonce, ptxt []byte, err error) {
	b := make([]byte, totalOH+512)
	n, err := f.f.ReadAt(b, 0)
	switch err {
	case nil:
		// This might indicate that the refresh token is longer than 512 bytes
		// (unlikely) or the file has had data appended to it. It might be
		// worth it to fail now so we can see the problem, but for now just let
		// AEAD fail.
	case io.EOF:
		// Expected case. Do nothing.
	default:
		return nil, nil, fmt.Errorf("couldn't read token file contents: %w", err)
	}
	b = b[:n]
	if len(b) == 0 {
		// File is empty. Load won't care; Store will set it up.
		return nil, nil, nil
	}
	if len(b) < totalOH {
		return nil, nil, errors.New("stored data is too short")
	}
	nonce = b[:nonceSize]
	text := b[nonceSize:]
	ptxt, err = f.enc.Open(text[:0], nonce, text, nil)
	if err != nil {
		return nil, nil, err
	}
	return nonce, ptxt, nil
}

func initialNonce(rt string, rand io.Reader) []byte {
	b := make([]byte, nonceSize, totalOH+len(rt))
	pad := b[8:nonceSize]
	_, err := io.ReadFull(rand, pad)
	if err != nil {
		panic(fmt.Errorf("couldn't read nonce padding: %w", err))
	}
	return b
}
