package brain_test

import (
	"math/rand/v2"
	"testing"

	"github.com/zephyrtronium/robot/brain"
)

func TestSkip(t *testing.T) {
	var counts [11]int // small array so i can use rejection criterion from a table
	const N = 50000
	for range N {
		var s brain.Skip
		var k int
		for {
			d := int(s.N(rand.Uint64(), rand.Uint64()))
			// Increment the skip size to represent accepting a term.
			d++
			if k+d >= len(counts) {
				break
			}
			k += d
		}
		counts[k]++
	}
	var x2 float64
	for _, o := range counts {
		x2 += float64(o * o)
	}
	x2 /= N / float64(len(counts))
	x2 -= N
	if x2 > 29.59 {
		t.Errorf("uniformity rejected at p=0.001 level with statistic %.3f\n%d", x2, counts)
	}
}

func BenchmarkSkip(b *testing.B) {
	var s brain.Skip
	for range b.N {
		s.N(rand.Uint64(), rand.Uint64())
	}
}
