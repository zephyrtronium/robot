package channel_test

import (
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/zephyrtronium/robot/channel"
)

func TestHistory(t *testing.T) {
	h := channel.NewHistory()
	var wg sync.WaitGroup
	wg.Add(1 << 11)
	for i := range 1 << 11 {
		go func() {
			defer wg.Done()
			seen := make(map[string]bool)
			h.Add(strconv.Itoa(i), strconv.Itoa(i), strconv.Itoa(i))
			for who, text := range h.All() {
				if who == "" || text == "" {
					t.Errorf("empty iter: who=%q text=%q", who, text)
					return
				}
				if who != text {
					t.Errorf("inconsistent: who=%q text=%q", who, text)
				}
				if seen[who] {
					t.Errorf("repeated %q", who)
					return
				}
				seen[who] = true
			}
			if len(seen) == 0 {
				t.Errorf("too few iters: %d", len(seen))
			}
		}()
	}
	wg.Wait()
}

func TestHistoryRange(t *testing.T) {
	// NOTE(zeph): this test depends on ring buffers being size 512
	t.Run("empty", func(t *testing.T) {
		h := channel.NewHistory()
		var got []string
		for who := range h.All() {
			got = append(got, who)
		}
		if len(got) != 0 {
			t.Errorf("too many iters: got %v, want empty", got)
		}
	})
	t.Run("short", func(t *testing.T) {
		h := channel.NewHistory()
		h.Add("1", "1", "1")
		var got []string
		for who := range h.All() {
			got = append(got, who)
		}
		want := []string{"1"}
		if !slices.Equal(got, want) {
			t.Errorf("wrong iters: want %v, got %v", want, got)
		}
	})
	t.Run("even", func(t *testing.T) {
		h := channel.NewHistory()
		var want []string
		// Iteration count here must match the size of the ring buffer.
		for i := range 512 {
			s := strconv.Itoa(i)
			want = append(want, s)
			h.Add(s, s, s)
		}
		var got []string
		for who := range h.All() {
			got = append(got, who)
		}
		if !slices.Equal(want, got) {
			t.Errorf("wrong iters: want %v\n got %v (len %d)", want, got, len(got))
		}
	})
	t.Run("long", func(t *testing.T) {
		h := channel.NewHistory()
		var want []string
		for i := range 600 {
			s := strconv.Itoa(i)
			want = append(want, s)
			h.Add(s, s, s)
		}
		// Slice here must match the size of the ring buffer.
		want = want[len(want)-512:]
		var got []string
		for who := range h.All() {
			got = append(got, who)
		}
		if !slices.Equal(want, got) {
			t.Errorf("wrong iters: want %v\n got %v (len %d)", want, got, len(got))
		}
	})
}
