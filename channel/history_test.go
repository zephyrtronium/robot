package channel_test

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/zephyrtronium/robot/channel"
)

func TestHistory(t *testing.T) {
	var h channel.History[int]
	var wg sync.WaitGroup
	const N = 100
	wg.Add(N)
	for i := range N {
		go func() {
			defer wg.Done()
			seen := make(map[int]bool)
			h.Add(time.Unix(int64(i)*60, 0), i)
			for m := range h.All() {
				seen[m] = true
			}
			if len(seen) == 0 {
				t.Errorf("too few iters: %d", len(seen))
			}
		}()
	}
	wg.Wait()
}

func TestHistoryAdd(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var h channel.History[int]
		got := slices.Collect(h.All())
		if len(got) != 0 {
			t.Errorf("too many iters: got %v, want empty", got)
		}
	})
	t.Run("short", func(t *testing.T) {
		var h channel.History[int]
		h.Add(time.Unix(1, 0), 1)
		got := slices.Collect(h.All())
		want := []int{1}
		if !slices.Equal(got, want) {
			t.Errorf("wrong iters: want %v, got %v", want, got)
		}
	})
	t.Run("drop", func(t *testing.T) {
		var h channel.History[int]
		h.Add(time.Unix(1, 0), 1)
		h.Add(time.Unix(1e6, 0), 2)
		got := slices.Collect(h.All())
		want := []int{2}
		if !slices.Equal(got, want) {
			t.Errorf("wrong iters: want %v, got %v", want, got)
		}
	})
}

func TestHistoryAllAtOnce(t *testing.T) {
	for range 100 {
		var h channel.History[int]
		ch := make(chan struct{})
		var wg sync.WaitGroup
		const N = 100
		wg.Add(N)
		for i := range N {
			go func() {
				defer wg.Done()
				<-ch
				h.Add(time.Unix(0, int64(i)), i)
			}()
		}
		close(ch)
		wg.Wait()
		got := slices.Collect(h.All())
		if len(got) != N {
			t.Errorf("wrong number of messages: want %d, got %d", N, len(got))
		}
	}
}
