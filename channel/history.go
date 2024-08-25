package channel

import "sync"

// History is a message history.
type History struct {
	mu   sync.Mutex
	ring []histelem
	k    uint64
}

type histelem struct {
	mu   sync.Mutex
	k    uint64 // total number of elements written up to this one
	id   string
	who  string
	text string
}

// ringsize is the number of messages in a history.
const ringsize = 1 << 9

// ringsize must be a power of 2; this line enforces that.
var _ [0]struct{} = [ringsize & (ringsize - 1)]struct{}{}

func NewHistory() *History {
	return &History{ring: make([]histelem, ringsize)}
}

func (h *History) Add(id, who, text string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	k := h.k % ringsize
	h.ring[k].mu.Lock()
	h.ring[k].k = k
	h.ring[k].id = id
	h.ring[k].who = who
	h.ring[k].text = text
	h.ring[k].mu.Unlock()
	h.k++ // We don't modulo so that All can detect changed elements.
}

// HistoryMessage is the minimal representation of a message recorded in a
// channel's history.
type HistoryMessage struct {
	ID     string
	Sender string
	Text   string
}

// Messages returns a slice of the messages in the channel history,
// approximately in order from oldest to newest.
func (h *History) Messages() []HistoryMessage {
	r := make([]HistoryMessage, 0, ringsize)
	h.mu.Lock()
	k := h.k
	// Iterate from ringsize tickets back.
	l := uint64(max(int64(k)-ringsize, 0))
	h.ring[l%ringsize].mu.Lock()
	h.mu.Unlock()
	for l < k {
		if h.ring[l%ringsize].k > k || h.ring[l%ringsize].who == "" {
			// Extra exit conditions.
			// We are currently holding the lock on ring[l%ringsize].
			// Set our final index to l so that we unlock it after the loop.
			k = l
			break
		}
		m := HistoryMessage{
			ID:     h.ring[l%ringsize].id,
			Sender: h.ring[l%ringsize].who,
			Text:   h.ring[l%ringsize].text,
		}
		r = append(r, m)
		// Lock the next element before we unlock the current one
		// so that no writer can skip past us.
		i := l + 1
		h.ring[i%ringsize].mu.Lock()
		h.ring[l%ringsize].mu.Unlock()
		l = i
	}
	h.ring[k%ringsize].mu.Unlock()
	return r
}
