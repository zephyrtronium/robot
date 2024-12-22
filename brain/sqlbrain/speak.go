package sqlbrain

import (
	"context"
	"fmt"
	"iter"

	"zombiezen.com/go/sqlite"
)

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
