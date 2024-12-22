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
)

// membrain is an implementation of braintest.Interface using in-memory maps
// to verify that the integration tests test the correct things.
type membrain struct {
	mu   sync.Mutex
	tups map[string]memtag
}

type memtag struct {
	tups    map[string][][2]string // map of prefixes to id and suffix
	forgort map[string]bool        // set of forgorten ids
}

var _ brain.Interface = (*membrain)(nil)

func (m *membrain) Learn(ctx context.Context, tag string, msg *brain.Message, tuples []brain.Tuple) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tups[tag].tups == nil {
		if m.tups == nil {
			m.tups = make(map[string]memtag)
		}
		m.tups[tag] = memtag{tups: make(map[string][][2]string), forgort: make(map[string]bool)}
	}
	r := m.tups[tag]
	for _, tup := range tuples {
		p := strings.Join(tup.Prefix, "\xff")
		r.tups[p] = append(r.tups[p], [2]string{msg.ID, tup.Suffix})
	}
	return nil
}

func (m *membrain) Forget(ctx context.Context, tag, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tups[tag].forgort == nil {
		m.tups[tag] = memtag{tups: make(map[string][][2]string), forgort: make(map[string]bool)}
	}
	m.tups[tag].forgort[id] = true
	return nil
}

func (m *membrain) Recall(ctx context.Context, tag string, page string, out []brain.Message) (n int, next string, err error) {
	panic("unimplemented")
}

func (m *membrain) Think(ctx context.Context, tag string, prompt []string) iter.Seq[func(id *[]byte, suf *[]byte) error] {
	return func(yield func(func(id *[]byte, suf *[]byte) error) bool) {
		m.mu.Lock()
		r := m.tups[tag]
		p := strings.Join(prompt, "\xff")
		// Copy out the values we'll yield so that we don't need to dance
		// around locks.
		y := slices.Clone(r.tups[p])
		m.mu.Unlock()
		for _, v := range y {
			// Check deletions during iteration so we don't unforget if the
			// iterating loop forgets things.
			m.mu.Lock()
			ok := m.tups[tag].forgort[v[0]]
			m.mu.Unlock()
			if ok {
				continue
			}
			f := func(id, suf *[]byte) error {
				*id = append(*id, v[0]...)
				*suf = append(*suf, v[1]...)
				return nil
			}
			if !yield(f) {
				break
			}
		}
	}
}

func (m *membrain) Speak(ctx context.Context, tag string, prompt []string, w *brain.Builder) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var s string
	if len(prompt) == 0 {
		u := slices.Clone(m.tups[tag].tups[""])
		d := 0
		for k, v := range u {
			if m.tups[tag].forgort[v[0]] {
				u[d], u[k] = u[k], u[d]
				d++
			}
		}
		u = u[d:]
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
		u := slices.Clone(m.tups[tag].tups[s])
		d := 0
		for k, v := range u {
			if m.tups[tag].forgort[v[0]] {
				u[d], u[k] = u[k], u[d]
				d++
			}
		}
		u = u[d:]
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
