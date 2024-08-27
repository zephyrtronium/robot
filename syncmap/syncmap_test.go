package syncmap

import (
	"sync"
	"testing"
	"testing/quick"
	"time"
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
func TestMap_All(t *testing.T) {
	m := New[string, int]()

	// Test empty map
	count := 0
	for range m.All() {
		count++
	}
	if count != 0 {
		t.Errorf("Expected 0 elements in empty map, got %d", count)
	}

	// Test with elements
	testData := map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	}

	for k, v := range testData {
		m.Store(k, v)
	}

	foundItems := make(map[string]int)
	for k, v := range m.All() {
		foundItems[k] = v
	}

	if len(foundItems) != len(testData) {
		t.Errorf("Expected %d elements, got %d", len(testData), len(foundItems))
	}

	for k, v := range testData {
		if foundV, ok := foundItems[k]; !ok || foundV != v {
			t.Errorf("Missing or incorrect value for key %s: expected %d, got %d", k, v, foundV)
		}
	}

	// Test early termination
	count = 0
	m.All()(func(k string, v int) bool {
		count++
		return count < 2
	})
	if count != 2 {
		t.Errorf("Expected early termination after 2 elements, got %d", count)
	}

	// Test concurrent modification
	go func() {
		m.Store("four", 4)
		m.Delete("two")
	}()

	time.Sleep(10 * time.Millisecond)

	foundItems = make(map[string]int)
	for k, v := range m.All() {
		foundItems[k] = v
	}

	if len(foundItems) < 2 || len(foundItems) > 4 {
		t.Errorf("Unexpected number of elements after concurrent modification: %d", len(foundItems))
	}
}

func TestMap_All_Quick(t *testing.T) {
	f := func(entries map[string]int) bool {
		m := New[string, int]()
		for k, v := range entries {
			m.Store(k, v)
		}

		seen := make(map[string]int)
		for k, v := range m.All() {
			seen[k] = v
		}

		if len(seen) != len(entries) {
			return false
		}

		for k, v := range entries {
			if seen[k] != v {
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
