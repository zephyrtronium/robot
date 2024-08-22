// Package deque provides a slice-backed double-ended queue.
package deque

import "slices"

// Deque is a slice-backed double-ended queue.
type Deque[Elem any] struct {
	el []Elem
	// left is the position of the leftmost valid element in el.
	// left >= len(el) implies the ring is empty.
	left int
}

// Len returns the number of elements in the deque.
func (d Deque[Elem]) Len() int {
	return len(d.el) - d.left
}

// Append adds elements tot he end of the deque.
func (d Deque[Elem]) Append(ee ...Elem) Deque[Elem] {
	d.el = append(d.el, ee...)
	return d
}

// Prepend adds elements to the front of the deque.
func (d Deque[Elem]) Prepend(ee ...Elem) Deque[Elem] {
	d = d.GrowFront(len(ee))
	d.left -= len(ee)
	copy(d.Slice(), ee)
	return d
}

// GrowFront ensures there is space to [Prepend] at least n elements.
func (d Deque[Elem]) GrowFront(n int) Deque[Elem] {
	if d.left >= n {
		return d
	}
	// Grow the slice, then slide the existing elements to the end.
	k := d.Len()
	d.el = slices.Grow(d.el, n)
	copy(d.el[cap(d.el)-k:cap(d.el)], d.Slice())
	d.el = d.el[:cap(d.el)]
	d.left = cap(d.el) - k
	return d
}

// GrowEnd ensures there is space to [Append] at least n elements.
func (d Deque[Elem]) GrowEnd(n int) Deque[Elem] {
	d.el = slices.Grow(d.el, n)
	return d
}

// DropEnd removes n elements from the end of the deque.
// If n is negative, there is no change.
// If n is larger than the deque's size, the result is empty.
func (d Deque[Elem]) DropEnd(n int) Deque[Elem] {
	if n <= 0 {
		return d
	}
	if n >= d.Len() {
		return d.Reset()
	}
	d.el = d.el[:len(d.el)-n]
	return d
}

// DropEndWhile removes elements from the end of the deque until the predicate
// returns false.
// The pointer passed to the predicate is a view into the deque's memory.
func (d Deque[Elem]) DropEndWhile(pred func(Elem) bool) Deque[Elem] {
	for len(d.el) > d.left {
		if !pred(d.el[len(d.el)-1]) {
			break
		}
		d.el = d.el[:len(d.el)-1]
	}
	return d
}

// Reset removes all elements from the deque.
func (d Deque[Elem]) Reset() Deque[Elem] {
	d.left = len(d.el)
	return d
}

// Slice returns a view into the deque's memory.
// Elements prepended to the deque appear at the beginning of the slice.
func (d Deque[Elem]) Slice() []Elem {
	return d.el[d.left:]
}
