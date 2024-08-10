package sqlbrain

import (
	"context"
	"fmt"
	"math/rand/v2"

	"zombiezen.com/go/sqlite"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/prepend"
	"github.com/zephyrtronium/robot/tpool"
)

var prependerPool tpool.Pool[*prepend.List[string]]

// Speak generates a full message and appends it to w.
// The prompt is in reverse order and has entropy reduction applied.
func (br *Brain) Speak(ctx context.Context, tag string, prompt []string, w []byte) ([]byte, error) {
	search := prependerPool.Get().Set(prompt...)
	defer func() { prependerPool.Put(search) }()

	conn, err := br.db.Take(ctx)
	defer br.db.Put(conn)
	if err != nil {
		return w, fmt.Errorf("couldn't get connection to speak: %w", err)
	}

	b := make([]byte, 0, 128)
	for range 1024 {
		var err error
		var l int
		b, l, err = next(conn, tag, b, search.Slice())
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			break
		}
		w = append(w, b...)
		search = search.Drop(search.Len() - l - 1).Prepend(brain.ReduceEntropy(string(b)))
	}
	return w, nil
}

func next(conn *sqlite.Conn, tag string, b []byte, prompt []string) ([]byte, int, error) {
	if len(prompt) == 0 {
		var err error
		b, err = first(conn, tag, b)
		return b, 0, err
	}
	st, err := conn.Prepare(`SELECT suffix FROM knowledge WHERE tag = :tag AND prefix >= :lower AND prefix < :upper AND LIKELY(deleted IS NULL)`)
	if err != nil {
		return b[:0], len(prompt), fmt.Errorf("couldn't prepare term selection: %w", err)
	}
	st.SetText(":tag", tag)
	w := make([]byte, 0, 32)
	var d []byte
	var skip brain.Skip
	picked := 0
	for {
		b = prefix(b[:0], prompt)
		b, d = searchbounds(b)
		st.SetBytes(":lower", b)
		st.SetBytes(":upper", d)
	sel:
		for {
			ok, err := st.Step()
			if err != nil {
				return b[:0], len(prompt), fmt.Errorf("couldn't step term selection: %w", err)
			}
			if !ok {
				break
			}
			n := st.ColumnLen(0)
			if cap(w) < n {
				w = make([]byte, n)
			}
			w = w[:st.ColumnBytes(0, w[:n])]
			picked++
			for range skip.N(rand.Uint64(), rand.Uint64()) {
				ok, err := st.Step()
				if err != nil {
					return b[:0], len(prompt), fmt.Errorf("couldn't step term selection: %w", err)
				}
				if !ok {
					break sel
				}
			}
		}
		if picked < 3 && len(prompt) > 1 {
			// We haven't seen enough options, and we have context we could
			// lose. Do so and try again from the beginning.
			prompt = prompt[:len(prompt)-1]
			if err := st.Reset(); err != nil {
				return b[:0], len(prompt), fmt.Errorf("couldn't reset term selection: %w", err)
			}
			continue
		}
		// Note that this also handles the case where there were no results.
		b = append(b[:0], w...)
		return b, len(prompt), nil
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

func first(conn *sqlite.Conn, tag string, b []byte) ([]byte, error) {
	b = b[:0] // in case we get no rows
	s, err := conn.Prepare(`SELECT suffix FROM knowledge WHERE tag = :tag AND prefix = x'' AND LIKELY(deleted IS NULL)`)
	if err != nil {
		return b, fmt.Errorf("couldn't prepare first term selection: %w", err)
	}
	s.SetText(":tag", tag)
	var skip brain.Skip
sel:
	for {
		ok, err := s.Step()
		if err != nil {
			return b[:0], fmt.Errorf("couldn't step first term selection: %w", err)
		}
		if !ok {
			break
		}
		n := s.ColumnLen(0)
		if cap(b) < n {
			b = make([]byte, n)
		}
		b = b[:s.ColumnBytes(0, b[:n])]
		for range skip.N(rand.Uint64(), rand.Uint64()) {
			ok, err := s.Step()
			if err != nil {
				return b[:0], fmt.Errorf("couldn't step first term selection: %w", err)
			}
			if !ok {
				break sel
			}
		}
	}
	return b, nil
}
