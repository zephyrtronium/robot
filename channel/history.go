package channel

import (
	"iter"
	"sync/atomic"
	"time"
)

// History is a message history.
type History struct {
	oldest, newest atomic.Pointer[histnode]
}

type histnode struct {
	newer atomic.Pointer[histnode]

	id   string
	who  string
	text string
	exp  int64
}

// sentinel is a special node indicating that the next link is being modified.
var sentinel = new(histnode)

func (h *History) Add(now time.Time, id, who, text string) {
	h.dropOld(now.UnixNano())
	l := &histnode{
		id:   id,
		who:  who,
		text: text,
		exp:  now.Add(15 * time.Minute).UnixNano(),
	}
	for {
		if h.oldest.CompareAndSwap(nil, sentinel) {
			// List was empty.
			h.newest.Store(l)
			h.oldest.Store(l)
			return
		}
		f := h.newest.Swap(sentinel)
		for f == sentinel {
			f = h.newest.Swap(sentinel)
		}
		if f == nil {
			// The list became empty while we were spinning. Retry.
			h.newest.CompareAndSwap(sentinel, nil)
			continue
		}
		f.newer.Store(l)
		h.newest.Store(l)
		return
	}
}

func (h *History) dropOld(exp int64) {
	for {
		cur := h.oldest.Swap(sentinel)
		for cur == sentinel {
			// Another goroutine is doing the same thing. Wait for them.
			cur = h.oldest.Swap(sentinel)
		}
		if cur == nil {
			// Cleared the list.
			h.newest.Store(nil)
			h.oldest.Store(nil)
			return
		}
		if cur.exp > exp {
			// The rest are current.
			h.oldest.Store(cur)
			return
		}
		// Store the next element back into the oldest position and load once
		// more in the next iteration. This way, if another goroutine is trying
		// to iterate, it gets to make progress.
		next := cur.newer.Load()
		h.oldest.Store(next)
	}
}

// HistoryMessage is the minimized representation of a message recorded in a
// channel's history.
type HistoryMessage struct {
	ID     string
	Sender string
	Text   string
}

// All yields the messages in the history, approximately in order from
// oldest to newest.
func (h *History) All() iter.Seq[HistoryMessage] {
	return func(yield func(HistoryMessage) bool) {
		p := &h.oldest
		for {
			cur := p.Load()
			for cur == sentinel {
				cur = p.Load()
			}
			if cur == nil {
				return
			}
			v := HistoryMessage{ID: cur.id, Sender: cur.who, Text: cur.text}
			if !yield(v) {
				return
			}
			p = &cur.newer
		}
	}
}
