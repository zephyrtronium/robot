package main

import (
	"fmt"
	"iter"
	"strings"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func ancientOpen(file string) (*sqlite.Conn, int, error) {
	conn, err := sqlite.OpenConn(file, sqlite.OpenReadOnly, sqlite.OpenURI)
	if err != nil {
		return nil, 0, err
	}
	st, _, err := conn.PrepareTransient(`SELECT pfix FROM config`)
	if err != nil {
		return nil, 0, err
	}
	defer st.Finalize()
	order, err := sqlitex.ResultInt(st)
	return conn, order, err
}

type ancientMessage struct {
	tag  string
	text string
}

func ancientMessages(conn *sqlite.Conn, order int) iter.Seq2[ancientMessage, error] {
	// don't look at this code
	return func(yield func(ancientMessage, error) bool) {
		st, _, err := conn.PrepareTransient(fmt.Sprintf(`SELECT id, tag, IFNULL(p%d, ''), IFNULL(suffix, '') FROM tuples%d`, order-1, order))
		if err != nil {
			yield(ancientMessage{}, err)
			return
		}
		defer st.Finalize()
		var (
			toks          []string
			id            int64
			tag, pre, suf string
		)
		for {
			// Get the initial values for the current message.
			for {
				ok, err := st.Step()
				if err != nil {
					yield(ancientMessage{}, err)
					return
				}
				if !ok {
					return
				}
				pre = st.ColumnText(2)
				if pre != "" {
					continue
				}
				id = st.ColumnInt64(0)
				tag = st.ColumnText(1)
				suf = st.ColumnText(3)
				break
			}
			toks = toks[:0]
			for {
				if suf == "" {
					if !yield(ancientMessage{tag: tag, text: strings.Join(toks, " ")}, nil) {
						return
					}
					break
				}
				toks = append(toks, suf)
				ok, err := st.Step()
				if err != nil {
					yield(ancientMessage{}, err)
					return
				}
				if !ok {
					yield(ancientMessage{tag: tag, text: strings.Join(toks, " ")}, nil)
					return
				}
				nid := st.ColumnInt64(0)
				if nid != id+1 {
					// Gap in IDs. Skip to the next message.
					for suf != "" {
						ok, err := st.Step()
						if err != nil {
							yield(ancientMessage{}, err)
							return
						}
						if !ok {
							return
						}
						suf = st.ColumnText(3)
					}
					break
				}
				id = nid
				suf = st.ColumnText(3)
			}
		}
	}
}
