package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/chacha20poly1305"
)

type buffer struct {
	b []byte
}

func (b *buffer) ReadAt(p []byte, off int64) (n int, err error) {
	if off > int64(len(b.b)) {
		return 0, io.EOF
	}
	n = copy(p, b.b[off:])
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}

func (b *buffer) WriteAt(p []byte, off int64) (n int, err error) {
	bl := int64(len(b.b))
	pl := int64(len(p))
	if bl < off+pl {
		b.b = append(b.b, make([]byte, pl+off-bl)...)
	}
	return copy(b.b[off:], p), nil
}

func (b *buffer) Truncate(n int64) error {
	b.b = b.b[:0]
	return nil
}

func TestLoad(t *testing.T) {
	cases := []struct {
		name string
		key  [KeySize]byte
		data []byte
		want string
		ok   bool
	}{
		{
			name: "empty",
			key:  [KeySize]byte{},
			data: nil,
			want: "",
			ok:   true,
		},
		{
			name: "ok",
			key:  [KeySize]byte{},
			data: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfd, 0x68, 0x84, 0xdd, 0x3d, 0x38, 0x9d, 0xd0, 0x31, 0xe7, 0x79, 0x67, 0xe0, 0xfc, 0x12, 0xb, 0x43, 0xe, 0x70, 0x5e, 0x55, 0x5},
			want: "bocchi",
			ok:   true,
		},
		{
			name: "wrong",
			key:  [KeySize]byte{0: 1},
			data: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfd, 0x68, 0x84, 0xdd, 0x3d, 0x38, 0x9d, 0xd0, 0x31, 0xe7, 0x79, 0x67, 0xe0, 0xfc, 0x12, 0xb, 0x43, 0xe, 0x70, 0x5e, 0x55, 0x5},
			want: "",
			ok:   false,
		},
		{
			name: "short",
			key:  [KeySize]byte{},
			data: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			want: "",
			ok:   false,
		},
		{
			name: "long",
			key:  [KeySize]byte{},
			data: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfd, 0x68, 0x84, 0xdd, 0x3d, 0x38, 0x9d, 0xd0, 0x31, 0xe7, 0x79, 0x67, 0xe0, 0xfc, 0x12, 0xb, 0x43, 0xe, 0x70, 0x5e, 0x55, 0x5, 0x0},
			want: "",
			ok:   false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := buffer{b: bytes.Clone(c.data)}
			enc, err := chacha20poly1305.New(c.key[:])
			if err != nil {
				panic(err)
			}
			s := FileStorage{f: &f, enc: enc}

			got, err := s.Load(context.Background())
			if (err == nil) != c.ok {
				t.Errorf("wrong error: compare to nil should be %t but got %v", c.ok, err)
			}
			if got != c.want {
				t.Errorf("wrong result:\nwant %q\ngot  %q", c.want, got)
			}
		})
	}
}

func TestStore(t *testing.T) {
	cases := []struct {
		name string
		key  [KeySize]byte
		pre  []byte
		data string
		want []byte
		ok   bool
	}{
		{
			name: "new",
			key:  [KeySize]byte{},
			pre:  nil,
			data: "bocchi",
			want: []byte{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2, 0x3, 0x4, 0x91, 0x2f, 0x20, 0x31, 0x58, 0xac, 0xb4, 0x18, 0xd9, 0xa2, 0xa8, 0x40, 0x43, 0xbd, 0x81, 0x59, 0xd0, 0x65, 0x2c, 0xea, 0xe0, 0x98},
			ok:   true,
		},
		{
			name: "clear",
			key:  [KeySize]byte{},
			pre:  []byte{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2, 0x3, 0x4, 0x91, 0x2f, 0x20, 0x31, 0x58, 0xac, 0xb4, 0x18, 0xd9, 0xa2, 0xa8, 0x40, 0x43, 0xbd, 0x81, 0x59, 0xd0, 0x65, 0x2c, 0xea, 0xe0, 0x98},
			data: "",
			want: nil,
			ok:   true,
		},
		{
			name: "change",
			key:  [KeySize]byte{},
			pre:  []byte{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2, 0x3, 0x4, 0x91, 0x2f, 0x20, 0x31, 0x58, 0xac, 0xb4, 0x18, 0xd9, 0xa2, 0xa8, 0x40, 0x43, 0xbd, 0x81, 0x59, 0xd0, 0x65, 0x2c, 0xea, 0xe0, 0x98},
			data: "nijika",
			want: []byte{0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2, 0x3, 0x4, 0x1c, 0x88, 0xd9, 0xee, 0x74, 0x39, 0xd5, 0xde, 0xa7, 0x49, 0x8a, 0xad, 0x35, 0xc7, 0x40, 0x4, 0x3c, 0x37, 0x88, 0x21, 0x6c, 0x8f},
			ok:   true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := buffer{b: bytes.Clone(c.pre)}
			enc, err := chacha20poly1305.New(c.key[:])
			if err != nil {
				panic(err)
			}
			s := FileStorage{f: &f, enc: enc, rand: bytes.NewReader([]byte{1, 2, 3, 4})}

			err = s.Store(context.Background(), c.data)
			if (err == nil) != c.ok {
				t.Errorf("wrong error: compare to nil should be %t but got %v", c.ok, err)
			}
			if !bytes.Equal(f.b, c.want) {
				t.Errorf("wrong stored buffer:\nwant %q\ngot  %q", c.want, f.b)
			}
		})
	}
}

func TestInitialNonce(t *testing.T) {
	// We're only interested in whether the reader is used in the right place,
	// so there isn't much reason to use a table test. There are other aspects
	// of the code that could be tested with more cases, but they aren't
	// semantic, and it isn't worth it to bake them in.
	b := bytes.NewReader([]byte{1, 2, 3, 4})
	want := []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}
	got := initialNonce("bocchi", b)
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
	if r != "" {
		t.Errorf("unexpected initial token: %q", r)
	}
	if err := s.Store(ctx, "bocchi"); err != nil {
		t.Errorf("error saving bocchi: %v", err)
	}
	r, err = s.Load(ctx)
	if err != nil {
		t.Errorf("couldn't load bocchi: %v", err)
	}
	if r != "bocchi" {
		t.Errorf("didn't load bocchi, instead %q", r)
	}
	if err := s.Store(ctx, ""); err != nil {
		t.Errorf("couldn't clear: %v", err)
	}
	r, err = s.Load(ctx)
	if err != nil {
		t.Errorf("couldn't load after clear: %v", err)
	}
	if r != "" {
		t.Errorf("didn't clear, instead %q", r)
	}
}
