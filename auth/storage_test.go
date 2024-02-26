package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/oauth2"
)

func TestInitialNonce(t *testing.T) {
	// We're only interested in whether the reader is used in the right place,
	// so there isn't much reason to use a table test. There are other aspects
	// of the code that could be tested with more cases, but they aren't
	// semantic, and it isn't worth it to bake them in.
	b := bytes.NewReader([]byte{1, 2, 3, 4})
	want := []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}
	got := initialNonce([]byte("bocchi"), b)
	if !bytes.Equal(want, got) {
		t.Errorf("wrong result:\nwant %v\ngot  %v", want, got)
	}
}

func TestFileStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("don't use filesystem in short testing")
	}
	d, err := os.MkdirTemp("", "robot-auth-file-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	p := filepath.Join(d, "test")
	key := [KeySize]byte{}
	if _, err := rand.Read(key[:]); err != nil {
		t.Fatal(err)
	}
	s, err := NewFileAt(p, key)
	if err != nil {
		t.Fatalf("couldn't open token file: %v", err)
	}
	ctx := context.Background()
	r, err := s.Load(ctx)
	if err != nil {
		t.Errorf("initial load error: %v", err)
	}
	if r != nil {
		t.Errorf("unexpected initial token: %#v", r)
	}
	tok := &oauth2.Token{AccessToken: "bocchi", RefreshToken: "ryou"}
	if err := s.Store(ctx, tok); err != nil {
		t.Errorf("error saving bocchi: %v", err)
	}
	r, err = s.Load(ctx)
	if err != nil {
		t.Errorf("couldn't load bocchi: %v", err)
	}
	if *r != *tok {
		t.Errorf("didn't load bocchi, instead %#v", r)
	}
	if err := s.Store(ctx, nil); err != nil {
		t.Errorf("couldn't clear: %v", err)
	}
	r, err = s.Load(ctx)
	if err != nil {
		t.Errorf("couldn't load after clear: %v", err)
	}
	if r != nil {
		t.Errorf("didn't clear, instead %#v", r)
	}
}
