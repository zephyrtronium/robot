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
			for _, m := range h.Messages() {
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

func TestHistoryRange(t *testing.T) {
	// NOTE(zeph): this test depends on ring buffers being size 512
	t.Run("empty", func(t *testing.T) {
		h := channel.NewHistory()
		got := h.Messages()
		if len(got) != 0 {
			t.Errorf("too many iters: got %v, want empty", got)
		}
	})
	t.Run("short", func(t *testing.T) {
		h := channel.NewHistory()
		h.Add("1", "1", "1")
		got := h.Messages()
		want := []channel.HistoryMessage{{ID: "1", Sender: "1", Text: "1"}}
		if !slices.Equal(got, want) {
			t.Errorf("wrong iters: want %v, got %v", want, got)
		}
	})
	t.Run("even", func(t *testing.T) {
		h := channel.NewHistory()
		var want []channel.HistoryMessage
		// Iteration count here must match the size of the ring buffer.
		for i := range 512 {
			s := strconv.Itoa(i)
			want = append(want, channel.HistoryMessage{ID: s, Sender: s, Text: s})
			h.Add(s, s, s)
		}
		got := h.Messages()
		if !slices.Equal(want, got) {
			t.Errorf("wrong iters: want %v\n got %v (len %d)", want, got, len(got))
		}
	})
	t.Run("long", func(t *testing.T) {
		h := channel.NewHistory()
		var want []channel.HistoryMessage
		for i := range 600 {
			s := strconv.Itoa(i)
			want = append(want, channel.HistoryMessage{ID: s, Sender: s, Text: s})
			h.Add(s, s, s)
		}
		// Slice here must match the size of the ring buffer.
		want = want[len(want)-512:]
		got := h.Messages()
		if !slices.Equal(want, got) {
			t.Errorf("wrong iters: want %v\n got %v (len %d)", want, got, len(got))
		}
	})
}
