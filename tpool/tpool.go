// Package tpool provides a generic, type-safe sync.Pool wrapper.
package tpool

import "sync"

// Pool is a type-safe wrapper around a [sync.Pool].
// To obtain one, declare a variable or convert an existing sync.Pool to it.
// In the latter case, if the pool's New field is non-nil,
// it must return values which assert to T.
type Pool[T any] sync.Pool

// Get pulls a value from the pool.
// It is a thin wrapper around [*sync.Pool.Get] and so mirrors its semantics.
// If the pool's New field is non-nil and returns a value which does not assert to T,
// then the result is the zero value of T.
func (p *Pool[T]) Get() T {
	r, _ := (*sync.Pool)(p).Get().(T)
	return r
}

// Put returns a value to the pool.
// It is a thin wrapper around [*sync.Pool.Put] and so mirrors its semantics.
func (p *Pool[T]) Put(e T) {
	(*sync.Pool)(p).Put(e)
}
