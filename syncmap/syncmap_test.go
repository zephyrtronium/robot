package syncmap

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"
	"testing/quick"
	"time"
)

// TestMap_All requires the race detector to be useful
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
	var editwg sync.WaitGroup
	var done = make(chan struct{})

	editwg.Add(1)
	go func() {
		defer editwg.Done()
		start := time.Now()
		for time.Since(start) < time.Second {
			switch rand.IntN(3) {
			case 0:
				m.Store(fmt.Sprintf("key%d", rand.IntN(100)), rand.IntN(1000))
			case 1:
				m.Delete(fmt.Sprintf("key%d", rand.IntN(100)))
			case 2:
				m.Store(fmt.Sprintf("key%d", rand.IntN(100)), rand.IntN(1000))
				m.Delete(fmt.Sprintf("key%d", rand.IntN(100)))
			}
		}
	}()

	go func() {
		for {
			for k, v := range m.All() {
				foundItems[k] = v // sink
			}

			select {
			case <-done:
				return
			default:
			}
		}
	}()

	editwg.Wait()
	close(done)
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
