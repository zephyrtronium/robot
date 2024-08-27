package channel_test

import (
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/zephyrtronium/robot/channel"
)

func TestHistory(t *testing.T) {
	var h channel.History
	var wg sync.WaitGroup
	const N = 100
	wg.Add(N)
	for i := range N {
		go func() {
			defer wg.Done()
			seen := make(map[string]bool)
			h.Add(time.Unix(int64(i)*60, 0), strconv.Itoa(i), strconv.Itoa(i), strconv.Itoa(i))
			for m := range h.All() {
				if m.Sender == "" || m.Text == "" {
					t.Errorf("empty iter: %q", m)
					return
				}
				if m.ID != m.Sender || m.ID != m.Text {
					t.Errorf("inconsistent: %q", m)
				}
				if seen[m.Sender] {
					t.Errorf("repeated %q", m.Sender)
					return
				}
				seen[m.Sender] = true
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
		var h channel.History
		got := slices.Collect(h.All())
		if len(got) != 0 {
			t.Errorf("too many iters: got %v, want empty", got)
		}
	})
	t.Run("short", func(t *testing.T) {
		var h channel.History
		h.Add(time.Unix(1, 0), "1", "1", "1")
		got := slices.Collect(h.All())
		want := []channel.HistoryMessage{{ID: "1", Sender: "1", Text: "1"}}
		if !slices.Equal(got, want) {
			t.Errorf("wrong iters: want %v, got %v", want, got)
		}
	})
	t.Run("drop", func(t *testing.T) {
		var h channel.History
		h.Add(time.Unix(1, 0), "1", "1", "1")
		h.Add(time.Unix(1e6, 0), "2", "2", "2")
		got := slices.Collect(h.All())
		want := []channel.HistoryMessage{{ID: "2", Sender: "2", Text: "2"}}
		if !slices.Equal(got, want) {
			t.Errorf("wrong iters: want %v, got %v", want, got)
		}
	})
}

func TestHistoryAllAtOnce(t *testing.T) {
	for range 100 {
		var h channel.History
		ch := make(chan struct{})
		var wg sync.WaitGroup
		const N = 100
		wg.Add(N)
		for i := range N {
			go func() {
				defer wg.Done()
				s := strconv.Itoa(i)
				<-ch
				h.Add(time.Unix(0, int64(i)), s, s, s)
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
