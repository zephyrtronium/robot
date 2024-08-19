package channel

import (
	_ "embed"
	"errors"
	"sync"
	"time"
)

// MemeDetector is literally a meme detector.
type MemeDetector struct {
	// mu guards the messages list and the counts.
	mu sync.Mutex
	// front is the node of the most recent message in history.
	front *node
	// back is the node of the oldest message in history.
	back *node
	// counts tracks the expiry time of each message said by each user.
	// Detected memes are recorded with the empty string as the user.
	counts map[string]map[string]int64 // map[message]map[user]UnixMillis

	// need is the number of messages needed to trigger memery.
	need int
	// within is the duration to hold messages.
	within time.Duration
}

// node is a node in a doubly linked list of messages sorted by time.
type node struct {
	older, newer *node

	msg  string
	user string
	exp  int64
}

// NewMemeDetector creates.
func NewMemeDetector(need int, within time.Duration) *MemeDetector {
	return &MemeDetector{
		counts: make(map[string]map[string]int64),
		need:   need,
		within: within,
	}
}

func (m *MemeDetector) chopLocked(now int64) {
	b := m.back
	// For each expired node at the back:
	for b != nil && b.exp <= now {
		// We will drop this node.
		// If the most recent expiry time from this user isn't newer than b,
		// we also need to stop tracking it in the map.
		if m.counts[b.msg][b.user] <= b.exp {
			if m.counts[b.msg] != nil {
				delete(m.counts[b.msg], b.user)
				// If there are no users left who sent this message,
				// stop tracking the message as well to control memory usage.
				if len(m.counts[b.msg]) == 0 {
					delete(m.counts, b.msg)
				}
			}
		}
		if b.newer == nil {
			m.front, m.back = nil, nil
			return
		}
		m.back = b.newer
		b.newer = nil // clear the pointer to help the gc
		b = m.back
		b.older = nil
	}
}

func (m *MemeDetector) insertLocked(msg, user string, exp int64) {
	new := &node{
		msg:  msg,
		user: user,
		exp:  exp,
	}
	if new.exp <= m.counts[msg][user] {
		// This message is (somehow) older than another from the same user.
		// We don't care about it.
		return
	}
	if m.counts[msg] == nil {
		m.counts[msg] = make(map[string]int64)
	}
	m.counts[msg][user] = new.exp

	if m.front == nil {
		m.front, m.back = new, new
		return
	}
	if new.exp >= m.front.exp {
		new.older = m.front
		m.front = new
		return
	}
	l := m.front
	for l.older != nil {
		if new.exp >= l.older.exp {
			new.newer, new.older = l, l.older
			l.older, l.older.newer = new, new
			return
		}
		l = l.older
	}
	new.newer = l
	l.older, m.back = new, new
}

// Check determines whether a message is a meme. If it is not, the returned
// error is NotCopypasta. Times passed to Check should be monotonic, as
// messages outside the detector's threshold are removed.
func (m *MemeDetector) Check(t time.Time, from, msg string) error {
	now := t.UnixMilli()
	m.mu.Lock()
	defer m.mu.Unlock()
	// Remove old messages and discard old memes.
	m.chopLocked(now)
	// Insert the new message.
	m.insertLocked(msg, from, now+m.within.Milliseconds())
	// Get the meme metric: number of distinct users who sent this message in
	// the time window.
	n := len(m.counts[msg])
	if n < m.need {
		return ErrNotCopypasta
	}
	// Genuine meme. But is it fresh?
	if _, ok := m.counts[msg][""]; ok {
		return ErrNotCopypasta
	}
	// It is, but not for the following fifteen minutes.
	m.insertLocked(msg, "", now+15*60*1000)
	return nil
}

// Block adds a message as a meme directly, preventing its reuse
// for fifteen minutes from t.
func (m *MemeDetector) Block(t time.Time, msg string) {
	now := t.UnixMilli()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.insertLocked(msg, "", now+15*60*1000)
}

// ErrNotCopypasta is a sentinel error returned by MemeDetector.Check when a
// message is not copypasta.
var ErrNotCopypasta = errors.New("not copypasta")
