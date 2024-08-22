package sqlbrain

import (
	"context"
	"fmt"
	"math/rand/v2"

	"zombiezen.com/go/sqlite"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/deque"
	"github.com/zephyrtronium/robot/tpool"
)

var prependerPool tpool.Pool[deque.Deque[string]]

// Speak generates a full message and appends it to w.
// The prompt is in reverse order and has entropy reduction applied.
func (br *Brain) Speak(ctx context.Context, tag string, prompt []string, w *brain.Builder) error {
	search := prependerPool.Get().Prepend(prompt...)
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
			picked++
			for range skip.N(rand.Uint64(), rand.Uint64()) {
				ok, err := st.Step()
				if err != nil {
					return b[:0], "", len(prompt), fmt.Errorf("couldn't step term selection: %w", err)
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
	s, err := conn.Prepare(`SELECT id, suffix FROM knowledge WHERE tag = :tag AND prefix = x'' AND LIKELY(deleted IS NULL)`)
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
