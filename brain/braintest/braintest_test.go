package braintest_test

import (
	"context"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/braintest"
	"github.com/zephyrtronium/robot/userhash"
)

// membrain is an implementation of braintest.Interface using in-memory maps
// to verify that the integration tests test the correct things.
type membrain struct {
	mu    sync.Mutex
	tups  map[string]map[string][]string       // map of tags to map of prefixes to suffixes
	users map[userhash.Hash][][3]string        // map of hashes to tag and prefix+suffix
	ids   map[string]map[uuid.UUID][][2]string // map of tags to map of ids to prefix+suffix
	tms   map[string]map[int64][][2]string     // map of tags to map of timestamps to prefix+suffix
}

var _ braintest.Interface = (*membrain)(nil)

func (m *membrain) Learn(ctx context.Context, tag string, user userhash.Hash, id uuid.UUID, t time.Time, tuples []brain.Tuple) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tups == nil {
		m.tups = make(map[string]map[string][]string)
		m.users = make(map[userhash.Hash][][3]string)
		m.ids = make(map[string]map[uuid.UUID][][2]string)
		m.tms = make(map[string]map[int64][][2]string)
	}
	if m.tups[tag] == nil {
		m.tups[tag] = make(map[string][]string)
	}
	r := m.tups[tag]
	if m.ids[tag] == nil {
		m.ids[tag] = make(map[uuid.UUID][][2]string)
	}
	ids := m.ids[tag]
	if m.tms[tag] == nil {
		m.tms[tag] = make(map[int64][][2]string)
	}
	tms := m.tms[tag]
	for _, tup := range tuples {
		p := strings.Join(tup.Prefix, "\xff")
		r[p] = append(r[p], tup.Suffix)
		m.users[user] = append(m.users[user], [3]string{tag, p, tup.Suffix})
		ids[id] = append(ids[id], [2]string{p, tup.Suffix})
		tms[t.UnixNano()] = append(tms[t.UnixNano()], [2]string{p, tup.Suffix})
	}
	return nil
}

func (m *membrain) forgetLocked(tag, p, s string) {
	u := m.tups[tag][p]
	k := slices.Index(u, s)
	if k < 0 {
		return
	}
	u[k], u[len(u)-1] = u[len(u)-1], u[k]
	m.tups[tag][p] = u[:len(u)-1]
}

func (m *membrain) Forget(ctx context.Context, tag string, tuples []brain.Tuple) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, tup := range tuples {
		p := strings.Join(tup.Prefix, "\xff")
		m.forgetLocked(tag, p, tup.Suffix)
	}
	return nil
}

func (m *membrain) ForgetMessage(ctx context.Context, tag string, msg uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.ids[tag][msg]
	for _, v := range u {
		m.forgetLocked(tag, v[0], v[1])
	}
	delete(m.ids[tag], msg)
	return nil
}

func (m *membrain) ForgetDuring(ctx context.Context, tag string, since, before time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, b := since.UnixNano(), before.UnixNano()
	for tm, u := range m.tms[tag] {
		if tm < s || tm > b {
			continue
		}
		for _, v := range u {
			m.forgetLocked(tag, v[0], v[1])
		}
		delete(m.tms[tag], tm) // yea i modify the map during iteration, yea i'm cool
	}
	return nil
}

func (m *membrain) ForgetUser(ctx context.Context, user *userhash.Hash) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range m.users[*user] {
		m.forgetLocked(v[0], v[1], v[2])
	}
	delete(m.users, *user)
	return nil
}

func (m *membrain) Speak(ctx context.Context, tag string, prompt []string, w []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var s string
	if len(prompt) == 0 {
		u := m.tups[tag][""]
		if len(u) == 0 {
			return nil, nil
		}
		t := u[rand.IntN(len(u))]
		w = append(w, t...)
		w = append(w, ' ')
		s = brain.ReduceEntropy(t)
	} else {
		s = brain.ReduceEntropy(prompt[len(prompt)-1])
	}
	for range 256 {
		u := m.tups[tag][s]
		if len(u) == 0 {
			break
		}
		t := u[rand.IntN(len(u))]
		if t == "" {
			break
		}
		w = append(w, t...)
		w = append(w, ' ')
		s = brain.ReduceEntropy(t)
	}
	return w, nil
}

func TestTests(t *testing.T) {
	braintest.Test(context.Background(), t, func(ctx context.Context) braintest.Interface { return new(membrain) })
}
