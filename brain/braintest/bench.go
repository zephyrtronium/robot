package braintest

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

func randid() string {
	return strconv.FormatUint(rand.Uint64(), 16) + "-" + strconv.FormatUint(rand.Uint64(), 16)
}

// BenchLearn runs benchmarks on the brain's speed with recording new tuples.
// The learner returned by new must be safe for concurrent use.
func BenchLearn(ctx context.Context, b *testing.B, new func(ctx context.Context, b *testing.B) brain.Learner, cleanup func(brain.Learner)) {
	b.Run("similar", func(b *testing.B) {
		l := new(ctx, b)
		if cleanup != nil {
			b.Cleanup(func() { cleanup(l) })
		}
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			var t int64
			toks := []string{
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
			}
			for pb.Next() {
				t++
				toks[len(toks)-1] = strconv.FormatInt(t, 10)
				id := randid()
				u := userhash.Hash(randbytes(make([]byte, len(userhash.Hash{}))))
				err := brain.Learn(ctx, l, "bocchi", id, u, time.Unix(t, 0), strings.Join(toks, " "))
				if err != nil {
					b.Errorf("error while learning: %v", err)
				}
			}
		})
	})
	b.Run("distinct", func(b *testing.B) {
		l := new(ctx, b)
		if cleanup != nil {
			b.Cleanup(func() { cleanup(l) })
		}
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			var t int64
			toks := []string{
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
			}
			for pb.Next() {
				t++
				rand.Shuffle(len(toks), func(i, j int) { toks[i], toks[j] = toks[j], toks[i] })
				id := randid()
				u := userhash.Hash(randbytes(make([]byte, len(userhash.Hash{}))))
				err := brain.Learn(ctx, l, "bocchi", id, u, time.Unix(t, 0), strings.Join(toks[:8], " "))
				if err != nil {
					b.Errorf("error while learning: %v", err)
				}
			}
		})
	})
}

// BenchSpeak runs benchmarks on a brain's speed with generating messages
// from tuples. The brain returned by new must be safe for concurrent use.
func BenchSpeak(ctx context.Context, b *testing.B, new func(ctx context.Context, b *testing.B) brain.Brain, cleanup func(brain.Brain)) {
	sizes := []int64{1e3, 1e4, 1e5}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("similar-new-%d", size), func(b *testing.B) {
			br := new(ctx, b)
			if cleanup != nil {
				b.Cleanup(func() { cleanup(br) })
			}
			// First fill the brain.
			toks := []string{
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
			}
			for t := range size {
				toks[len(toks)-1] = strconv.FormatInt(t, 10)
				id := randid()
				u := userhash.Hash(randbytes(make([]byte, len(userhash.Hash{}))))
				err := brain.Learn(ctx, br, "bocchi", id, u, time.Unix(t, 0), strings.Join(toks, " "))
				if err != nil {
					b.Errorf("error while learning: %v", err)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if _, _, err := brain.Speak(ctx, br, "bocchi", ""); err != nil {
						b.Errorf("error while speaking: %v", err)
					}
				}
			})
		})
	}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("distinct-new-%d", size), func(b *testing.B) {
			br := new(ctx, b)
			if cleanup != nil {
				b.Cleanup(func() { cleanup(br) })
			}
			// First fill the brain.
			toks := []string{
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
			}
			for t := range size {
				rand.Shuffle(len(toks), func(i, j int) { toks[i], toks[j] = toks[j], toks[i] })
				id := randid()
				u := userhash.Hash(randbytes(make([]byte, len(userhash.Hash{}))))
				err := brain.Learn(ctx, br, "bocchi", id, u, time.Unix(t, 0), strings.Join(toks, " "))
				if err != nil {
					b.Errorf("error while learning: %v", err)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if _, _, err := brain.Speak(ctx, br, "bocchi", ""); err != nil {
						b.Errorf("error while speaking: %v", err)
					}
				}
			})
		})
	}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("distinct-prompted-%d", size), func(b *testing.B) {
			br := new(ctx, b)
			if cleanup != nil {
				b.Cleanup(func() { cleanup(br) })
			}
			// First fill the brain.
			toks := []string{
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
				hex.EncodeToString(randbytes(make([]byte, 4))),
			}
			for t := range size {
				rand.Shuffle(len(toks), func(i, j int) { toks[i], toks[j] = toks[j], toks[i] })
				id := randid()
				u := userhash.Hash(randbytes(make([]byte, len(userhash.Hash{}))))
				err := brain.Learn(ctx, br, "bocchi", id, u, time.Unix(t, 0), strings.Join(toks, " "))
				if err != nil {
					b.Errorf("error while learning: %v", err)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if _, _, err := brain.Speak(ctx, br, "bocchi", toks[rand.IntN(len(toks)-1)]); err != nil {
						b.Errorf("error while speaking: %v", err)
					}
				}
			})
		})
	}
}

// randbytes fills a slice of at least length 4 with random data.
func randbytes(b []byte) []byte {
	binary.NativeEndian.PutUint32(b, rand.Uint32())
	return b
}
