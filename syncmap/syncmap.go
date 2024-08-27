package syncmap

import (
	"iter"
	"sync"
)

// Map is a regular map but synchronized with a mutex.
type Map[K comparable, V any] struct {
	mu sync.Mutex
	m  map[K]V
}

// New returns a new syncmap.
func New[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		m: make(map[K]V),
	}
}

// Load returns the value for a key.
func (m *Map[K, V]) Load(key K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.m[key]
	return v, ok
}

// Store sets the value for a key.
func (m *Map[K, V]) Store(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[key] = value
}

// Delete deletes a key.
func (m *Map[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, key)
}

// Len returns the number of elements in the map.
func (m *Map[K, V]) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.m)
}

// All iterates over all elements in the map
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(f func(K, V) bool) {
		m.mu.Lock()
		for k, v := range m.m {
			m.mu.Unlock()
			if !f(k, v) {
				m.mu.Lock()
				break
			}

			m.mu.Lock()
		}

		m.mu.Unlock()
	}
}
