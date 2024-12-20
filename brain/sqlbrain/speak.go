package sqlbrain

import (
	"context"
	"fmt"
	"iter"
	"math/rand/v2"

	"zombiezen.com/go/sqlite"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/deque"
	"github.com/zephyrtronium/robot/tpool"
)

var prependerPool tpool.Pool[deque.Deque[string]]

func (br *Brain) Think(ctx context.Context, tag string, prompt []string) iter.Seq[func(id *[]byte, suf *[]byte) error] {
	return func(yield func(func(id, suf *[]byte) error) bool) {
		erf := func(err error) { yield(func(id, suf *[]byte) error { return err }) }
		conn, err := br.db.Take(ctx)
		defer br.db.Put(conn)
		if err != nil {
			erf(fmt.Errorf("couldn't get connection to speak: %w", err))
			return
		}
		var s *sqlite.Stmt
		if len(prompt) != 0 {
			s, err = conn.Prepare(`SELECT id, suffix FROM knowledge WHERE tag = :tag AND prefix >= :lower AND prefix < :upper AND LIKELY(deleted IS NULL)`)
			if err != nil {
				erf(fmt.Errorf("couldn't prepare term selection: %w", err))
				return
			}
			b := prefix(make([]byte, 0, 128), prompt)
			b, d := searchbounds(b)
			s.SetBytes(":lower", b)
			s.SetBytes(":upper", d)
		} else {
			s, err = conn.Prepare(`SELECT id, suffix FROM knowledge WHERE tag = :tag AND prefix = x'00' AND LIKELY(deleted IS NULL)`)
			if err != nil {
				erf(fmt.Errorf("couldn't prepare first term selection: %w", err))
				return
			}
		}
		s.SetText(":tag", tag)
		f := func(id, suf *[]byte) error {
			*id = bytecol(*id, s, 0)
			*suf = bytecol(*suf, s, 1)
			return nil
		}
		for {
			ok, err := s.Step()
			if err != nil {
				erf(fmt.Errorf("couldn't step term selection: %w", err))
				return
			}
			if !ok || !yield(f) {
				break
			}
		}
	}
}

func bytecol(d []byte, s *sqlite.Stmt, col int) []byte {
	n := s.ColumnLen(col)
	if cap(d) < n {
		d = make([]byte, n)
	}
	return d[:s.ColumnBytes(col, d[:n])]
}

// Speak generates a full message and appends it to w.
// The prompt is in reverse order and has entropy reduction applied.
func (br *Brain) Speak(ctx context.Context, tag string, prompt []string, w *brain.Builder) error {
	search := prependerPool.Get().Append("").Prepend(prompt...)
	defer func() { prependerPool.Put(search.Reset()) }()

	conn, err := br.db.Take(ctx)
	defer br.db.Put(conn)
	if err != nil {
		return fmt.Errorf("couldn't get connection to speak: %w", err)
	}

	b := make([]byte, 0, 128)
	for range 1024 {
		var err error
		var l int
		var id string
		b, id, l, err = next(conn, tag, b, search.Slice())
		if err != nil {
			return err
		}
		if len(b) == 0 {
			break
		}
		w.Append(id, b)
		search = search.DropEnd(search.Len() - l - 1).Prepend(brain.ReduceEntropy(string(b)))
	}
	return nil
}

func next(conn *sqlite.Conn, tag string, b []byte, prompt []string) ([]byte, string, int, error) {
	var id string
	if len(prompt) == 0 {
		var err error
		b, id, err = first(conn, tag, b)
		return b, id, 0, err
	}
	st, err := conn.Prepare(`SELECT id, suffix FROM knowledge WHERE tag = :tag AND prefix >= :lower AND prefix < :upper AND LIKELY(deleted IS NULL)`)
	if err != nil {
		return b[:0], "", len(prompt), fmt.Errorf("couldn't prepare term selection: %w", err)
	}
	st.SetText(":tag", tag)
	w := make([]byte, 0, 32)
	var (
		d    []byte
		skip brain.Skip
		t    uint64
	)
	for {
		var seen uint64
		b = prefix(b[:0], prompt)
		b, d = searchbounds(b)
		st.SetBytes(":lower", b)
		st.SetBytes(":upper", d)
	sel:
		for {
			for t > 0 {
				ok, err := st.Step()
				if err != nil {
					return b[:0], "", len(prompt), fmt.Errorf("couldn't step term selection: %w", err)
				}
				if !ok {
					break sel
				}
				seen++
				t--
			}
			ok, err := st.Step()
			if err != nil {
				return b[:0], "", len(prompt), fmt.Errorf("couldn't step term selection: %w", err)
			}
			if !ok {
				break
			}
			id = st.ColumnText(0)
			n := st.ColumnLen(1)
			if cap(w) < n {
				w = make([]byte, n)
			}
			w = w[:st.ColumnBytes(1, w[:n])]
			t = skip.N(rand.Uint64(), rand.Uint64())
		}
		// Try to lose context.
		// We want to do so when we have a long context and almost no options,
		// or at random with even a short context.
		// Note that in the latter case we use a 1/2 chance; it seems high, but
		// n.b. the caller will recover the last token that we discard.
		if len(prompt) > 4 && seen <= 2 || len(prompt) > 2 && rand.Uint32()&1 == 0 {
			prompt = prompt[:len(prompt)-1]
			if err := st.Reset(); err != nil {
				return b[:0], "", len(prompt), fmt.Errorf("couldn't reset term selection: %w", err)
			}
			continue
		}
		// Note that this also handles the case where there were no results.
		b = append(b[:0], w...)
		return b, id, len(prompt), nil
	}
}

// searchbounds produces the lower and upper bounds for a search by prefix.
// The upper bound is always a slice of the lower bound's underlying array.
func searchbounds(prefix []byte) (lower, upper []byte) {
	lower = append(prefix, prefix...)
	lower, upper = lower[:len(prefix)], lower[len(prefix):]
	if len(upper) != 0 {
		// The prefix is a list of terms each followed by a 0 byte.
		// So, the supremum of all strings with that prefix is the same with
		// the last byte replaced by 1.
		upper[len(upper)-1] = 1
	}
	return lower, upper
}

func first(conn *sqlite.Conn, tag string, b []byte) ([]byte, string, error) {
	var id string
	b = b[:0] // in case we get no rows
	s, err := conn.Prepare(`SELECT id, suffix FROM knowledge WHERE tag = :tag AND prefix = x'00' AND LIKELY(deleted IS NULL)`)
	if err != nil {
		return b, "", fmt.Errorf("couldn't prepare first term selection: %w", err)
	}
	s.SetText(":tag", tag)
	var skip brain.Skip
sel:
	for {
		ok, err := s.Step()
		if err != nil {
			return b[:0], "", fmt.Errorf("couldn't step first term selection: %w", err)
		}
		if !ok {
			break
		}
		id = s.ColumnText(0)
		n := s.ColumnLen(1)
		if cap(b) < n {
			b = make([]byte, n)
		}
		b = b[:s.ColumnBytes(1, b[:n])]
		for range skip.N(rand.Uint64(), rand.Uint64()) {
			ok, err := s.Step()
			if err != nil {
				return b[:0], "", fmt.Errorf("couldn't step first term selection: %w", err)
			}
			if !ok {
				break sel
			}
		}
	}
	return b, id, nil
}
