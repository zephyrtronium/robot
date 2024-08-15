package braintest_test

import (
	"context"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/braintest"
	"github.com/zephyrtronium/robot/userhash"
)

// membrain is an implementation of braintest.Interface using in-memory maps
// to verify that the integration tests test the correct things.
type membrain struct {
	mu    sync.Mutex
	tups  map[string]map[string][][2]string // map of tags to map of prefixes to id and suffix
	users map[userhash.Hash][][2]string     // map of hashes to tag and id
	tms   map[string]map[int64][]string     // map of tags to map of timestamps to ids
}

var _ brain.Brain = (*membrain)(nil)

func (m *membrain) Learn(ctx context.Context, tag, id string, user userhash.Hash, t time.Time, tuples []brain.Tuple) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tups[tag] == nil {
		if m.tups == nil {
			m.tups = make(map[string]map[string][][2]string)
			m.users = make(map[userhash.Hash][][2]string)
			m.tms = make(map[string]map[int64][]string)
		}
		m.tups[tag] = make(map[string][][2]string)
		m.tms[tag] = make(map[int64][]string)
	}
	m.users[user] = append(m.users[user], [2]string{tag, id})
	tms := m.tms[tag]
	tms[t.UnixNano()] = append(tms[t.UnixNano()], id)
	r := m.tups[tag]
	for _, tup := range tuples {
		p := strings.Join(tup.Prefix, "\xff")
		r[p] = append(r[p], [2]string{id, tup.Suffix})
	}
	return nil
}

func (m *membrain) forgetIDLocked(tag, id string) {
	for p, u := range m.tups[tag] {
		for len(u) > 0 {
			k := slices.IndexFunc(u, func(v [2]string) bool { return v[0] == id })
			if k < 0 {
				break
			}
			u[k], u[len(u)-1] = u[len(u)-1], u[k]
			u = u[:len(u)-1]
		}
		if len(u) != 0 {
			m.tups[tag][p] = u
		} else {
			delete(m.tups[tag], p)
		}
	}
}

func (m *membrain) Forget(ctx context.Context, tag string, tuples []brain.Tuple) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, tup := range tuples {
		p := strings.Join(tup.Prefix, "\xff")
		u := m.tups[tag][p]
		k := slices.IndexFunc(u, func(v [2]string) bool { return v[1] == tup.Suffix })
		if k < 0 {
			continue
		}
		u[k], u[len(u)-1] = u[len(u)-1], u[k]
		m.tups[tag][p] = u[:len(u)-1]
	}
	return nil
}

func (m *membrain) ForgetMessage(ctx context.Context, tag, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forgetIDLocked(tag, id)
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
			m.forgetIDLocked(tag, v)
		}
		delete(m.tms[tag], tm) // yea i modify the map during iteration, yea i'm cool
	}
	return nil
}

func (m *membrain) ForgetUser(ctx context.Context, user *userhash.Hash) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range m.users[*user] {
		m.forgetIDLocked(v[0], v[1])
	}
	delete(m.users, *user)
	return nil
}

func (m *membrain) Speak(ctx context.Context, tag string, prompt []string, w *brain.Builder) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var s string
	if len(prompt) == 0 {
		u := m.tups[tag][""]
		if len(u) == 0 {
			return nil
		}
		t := u[rand.IntN(len(u))]
		w.Append(t[0], []byte(t[1]))
		s = brain.ReduceEntropy(t[1])
	} else {
		s = brain.ReduceEntropy(prompt[len(prompt)-1])
	}
	for range 256 {
		u := m.tups[tag][s]
		if len(u) == 0 {
			break
		}
		t := u[rand.IntN(len(u))]
		if t[1] == "" {
			break
		}
		w.Append(t[0], []byte(t[1]))
		s = brain.ReduceEntropy(t[1])
	}
	return nil
}

func TestTests(t *testing.T) {
	braintest.Test(context.Background(), t, func(ctx context.Context) brain.Brain { return new(membrain) })
}
