package channel

import (
	"iter"
	"sync/atomic"
	"time"
	"unsafe"
)

// History is a history of recent messages.
// Entries automatically expire after fifteen minutes.
type History[M any] struct {
	oldest, newest atomic.Pointer[histnode[M]]
}

type histnode[M any] struct {
	newer atomic.Pointer[histnode[M]]
	msg   M
	exp   int64
}

var sentinel = unsafe.Pointer(new(byte))

func (h *History[M]) Add(now time.Time, msg M) {
	h.dropOld(now.UnixNano())
	l := &histnode[M]{
		msg: msg,
		exp: now.Add(15 * time.Minute).UnixNano(),
	}
	for {
		if h.oldest.CompareAndSwap(nil, (*histnode[M])(sentinel)) {
			// List was empty.
			h.newest.Store(l)
			h.oldest.Store(l)
			return
		}
		f := h.newest.Swap((*histnode[M])(sentinel))
		for f == (*histnode[M])(sentinel) {
			f = h.newest.Swap((*histnode[M])(sentinel))
		}
		if f == nil {
			// The list became empty while we were spinning. Retry.
			h.newest.CompareAndSwap((*histnode[M])(sentinel), nil)
			continue
		}
		f.newer.Store(l)
		h.newest.Store(l)
		return
	}
}

func (h *History[M]) dropOld(exp int64) {
	for {
		cur := h.oldest.Swap((*histnode[M])(sentinel))
		for cur == (*histnode[M])(sentinel) {
			// Another goroutine is doing the same thing. Wait for them.
			cur = h.oldest.Swap((*histnode[M])(sentinel))
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

// All yields the messages in the history, approximately in order from
// oldest to newest.
func (h *History[M]) All() iter.Seq[M] {
	return func(yield func(M) bool) {
		p := &h.oldest
		for {
			cur := p.Load()
			for cur == (*histnode[M])(sentinel) {
				cur = p.Load()
			}
			if cur == nil {
				return
			}
			if !yield(cur.msg) {
				return
			}
			p = &cur.newer
		}
	}
}
