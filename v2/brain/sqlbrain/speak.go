package sqlbrain

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strconv"

	"gitlab.com/zephyrtronium/sq"
)

// New creates a new prompt.
func (br *Brain) New(ctx context.Context, tag string) ([]string, error) {
	var s string
	err := br.stmts.newTuple.QueryRow(ctx, tag).Scan(&s)
	if err != nil {
		return nil, fmt.Errorf("couldn't get new chain: %w", err)
	}
	r := make([]string, br.order)
	r[br.order-1] = s
	return r, nil
}

// Speak creates a message from a prompt.
func (br *Brain) Speak(ctx context.Context, red func(string) string, tag string, prompt []string) ([]string, error) {
	names := make([]sq.NamedArg, 1+len(prompt))
	names[0] = sql.Named("tag", tag)
	terms := make([]sq.NullString, len(prompt))
	nn := 0
	for i, w := range prompt {
		nn += len(w) + 1
		terms[i] = sq.NullString{String: red(w), Valid: w != ""}
		names[i+1] = sql.Named("p"+strconv.Itoa(i), &terms[i])
	}
	p := make([]any, len(names))
	for i := range names {
		p[i] = names[i]
	}
	for nn < 500 {
		var w sq.NullString
		err := br.stmts.selectTuple.QueryRow(ctx, p...).Scan(&w)
		if err != nil {
			return nil, fmt.Errorf("couldn't scan chain with terms %v: %w", terms, err)
		}
		if !w.Valid {
			break
		}
		prompt = append(prompt, w.String)
		// Note that each p[i] is a named arg, and each name for prefix
		// elements aliases an element of terms. So, just updating terms is
		// sufficient to update the query parameters.
		copy(terms, terms[1:])
		terms[len(terms)-1] = w
	}
	return prompt, nil
}

//go:embed tuple.new.sql
var newTuple string

//go:embed tuple.select.sql
var selectTuple string
