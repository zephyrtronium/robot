// Package distro implements efficient random selection.
package distro

// Dist is a categorical distribution for efficient random selection.
type Dist[T any] struct {
	// cdf is the distribution. The last element is ^uint32(0).
	cdf []uint32
	// elems is the categories.
	elems []T
}

// New creates a cdf from a list of cases. Cases with non-positive weights are
// ignored. Otherwise, listing the same category in multiple cases is
// approximately equivalent to listing the category once with the sum of the
// positive weights.
func New[T any](c []Case[T]) *Dist[T] {
	d := Dist[T]{
		elems: make([]T, 0, len(c)),
	}
	var sum float64
	cum := make([]float64, 0, len(c))
	for _, c := range c {
		if c.W <= 0 {
			continue
		}
		sum += float64(c.W)
		cum = append(cum, sum)
		d.elems = append(d.elems, c.E)
	}
	d.cdf = make([]uint32, len(cum))
	for i, v := range cum {
		d.cdf[i] = uint32(v / sum * 0xffffffff)
	}
	if len(d.cdf) > 0 {
		// Guarantee cdf.
		d.cdf[len(d.cdf)-1] = 0xffffffff
	}
	return &d
}

// Pick selects the category corresponding to the given uniform variate. If
// the distribution is empty, the result is the zero value of T.
func (d *Dist[T]) Pick(v uint32) T {
	for i, u := range d.cdf {
		if v <= u {
			return d.elems[i]
		}
	}
	var zero T
	return zero
}

// Case is a category for a distribution.
type Case[T any] struct {
	// E is the category.
	E T
	// W is its relative weight.
	W int
}

// FromMap converts a map of categories to their weights to a slice of cases.
// Elements with non-positive weights are discarded.
func FromMap[T comparable](m map[T]int) []Case[T] {
	r := make([]Case[T], 0, len(m))
	for k, v := range m {
		if v > 0 {
			r = append(r, Case[T]{k, v})
		}
	}
	return r
}
