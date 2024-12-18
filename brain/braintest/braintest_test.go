package braintest_test

import (
	"context"
	"iter"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"

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

var _ brain.Interface = (*membrain)(nil)

func (m *membrain) Learn(ctx context.Context, tag string, msg *brain.Message, tuples []brain.Tuple) error {
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
	m.users[msg.Sender] = append(m.users[msg.Sender], [2]string{tag, msg.ID})
	tms := m.tms[tag]
	tms[msg.Timestamp] = append(tms[msg.Timestamp], msg.ID)
	r := m.tups[tag]
	for _, tup := range tuples {
		p := strings.Join(tup.Prefix, "\xff")
		r[p] = append(r[p], [2]string{msg.ID, tup.Suffix})
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

func (m *membrain) Forget(ctx context.Context, tag, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forgetIDLocked(tag, id)
	return nil
}

func (m *membrain) Recall(ctx context.Context, tag string, page string, out []brain.Message) (n int, next string, err error) {
	panic("unimplemented")
}

// Think implements brain.Interface.
func (m *membrain) Think(ctx context.Context, tag string, prefix []string) iter.Seq[func(id *[]byte, suf *[]byte) error] {
	panic("unimplemented")
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
	braintest.Test(context.Background(), t, func(ctx context.Context) brain.Interface { return new(membrain) })
}
