package prepend

// List is a list that minimizes copying while prepending.
// A nil *List is useful; methods which modify the list return a possibly new
// value, similar to the append builtin function.
type List[E any] struct {
	space []E
	k     int
}

// Len returns the number of elements in the list.
func (p *List[E]) Len() int {
	if p == nil {
		return 0
	}
	return len(p.space) - p.k
}

// Slice returns the elements in the list as a Slice directly into the list's
// owned memory.
func (p *List[E]) Slice() []E {
	if p == nil {
		return nil
	}
	return p.space[p.k:]
}

// Set sets the contents of the list.
func (p *List[E]) Set(ee ...E) *List[E] {
	if len(ee) == 0 {
		return p.Reset()
	}
	p = p.Reset()
	if len(ee) > len(p.space) {
		p.space = make([]E, len(ee))
	}
	p.k = len(p.space) - len(ee)
	copy(p.space[p.k:], ee)
	return p
}

// Prepend inserts elements in provided order at the start of the list.
func (p *List[E]) Prepend(ee ...E) *List[E] {
	if p == nil {
		p = new(List[E])
	}
	if p.k < len(ee) {
		// We don't expect enormous prompts, so a simple growth algorithm is fine.
		b := make([]E, cap(p.space)*2+len(ee))
		p.k = len(b) - len(p.space)
		copy(b[p.k:], p.space)
		p.space = b
	}
	p.k -= len(ee)
	copy(p.space[p.k:], ee)
	return p
}

// Drop removes the last n terms from the list.
// If n <= 0, there is no change.
// If n >= p.len(), the list becomes empty.
func (p *List[E]) Drop(n int) *List[E] {
	if n <= 0 {
		return p
	}
	if n >= p.Len() {
		// As a special case, we can reset the entire list when we drop all.
		// Note this branch also includes p == nil.
		return p.Reset()
	}
	p.space = p.space[:len(p.space)-n]
	return p
}

// Reset removes all elements from the list.
func (p *List[E]) Reset() *List[E] {
	if p == nil {
		return new(List[E])
	}
	p.space = p.space[:cap(p.space):cap(p.space)]
	p.k = cap(p.space)
	return p
}
