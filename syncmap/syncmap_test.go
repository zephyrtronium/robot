package syncmap

import (
	"sync"
	"testing"
)

func TestMap_Concurrent(t *testing.T) {
	m := New[int, int]()
	const goroutines = 100
	const operations = 10000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(base int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := base + j
				m.Store(key, key*2)
				if v, ok := m.Load(key); !ok || v != key*2 {
					t.Errorf("Concurrent operation failed: key=%d, expected=%d, got=%v", key, key*2, v)
				}
				m.Delete(key)
				if _, ok := m.Load(key); ok {
					t.Errorf("Delete operation failed: key=%d still exists", key)
				}
			}
		}(i * operations)
	}

	actualCount := 0
	for k, v := range m.All() {
		if v != k*2 {
			t.Errorf("Iteration mismatch: key=%d, expected=%d, got=%d", k, k*2, v)
		}
		actualCount++
	}

	wg.Wait()
}
