package brain

import "math"

// Skip computes numbers of elements to skip between samples to draw a single
// term uniformly from arbitrarily sized sequences.
//
// To draw a sample, create a new Skip, accept the first term of the sequence,
// skip N(rand(), rand()) terms, accept the next, and repeat N followed by an
// accept until the sequence is exhausted.
//
// Conceptually, drawing a sample from an arbitrary sequence can be performed
// by assigning each term a random weight and selecting the term that receives
// the largest.
// Skip transforms this procedure into modeling the number of
// random numbers it would take to find the next larger weight.
// Using this approach allows uniform sampling with O(log n) random numbers
// instead of O(n).
type Skip struct {
	u float64
}

// N computes the next skip length given two uniformly distributed random uint64s.
// The typical call will look like:
//
//	s.N(rand.Uint64(), rand.Uint64())
func (s *Skip) N(a, b uint64) uint64 {
	// We implement "Algorithm L" as described at
	// https://en.wikipedia.org/w/index.php?title=Reservoir_sampling&oldid=1237583224#Optimal:_Algorithm_L
	// with an adjustment: Algorithm L uses a term that decreases toward zero,
	// but we can instead make it increase toward 1 so that the zero value of a
	// Skip becomes useful.
	// Technically this can cause us to lose precision such that we are
	// no longer uniform over sequences with more than 2^53 terms.
	// I think that's an acceptable cost, as just iterating an empty loop that
	// many times would take unacceptably long, much less scanning a database.
	x := float64(a>>11) * 0x1p-53 // convert to [0, 1)
	y := float64(b>>11) * 0x1p-53
	// Again, Algorithm L updates the parameter after computing the skip length,
	// but we can do it beforehand so that an input value of 0 still does
	// the right thing.
	s.u = 1 - x*(1-s.u)
	return uint64(math.Log(y) / math.Log(s.u))
}
