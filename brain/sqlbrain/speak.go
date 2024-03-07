package sqlbrain

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"

	"gitlab.com/zephyrtronium/sq"

	"github.com/zephyrtronium/robot/brain"
)

func gumbelscan(rows *sq.Rows) (string, error) {
	var s string
	w := rand.Float64()
	if rows.Next() {
		err := rows.Scan(&s)
		if err != nil {
			return "", fmt.Errorf("couldn't scan first string in sample: %w", err)
		}
	}
Loop:
	for rows.Next() {
		u := math.Log(rand.Float64())/math.Log(1-w) + 1
		if math.IsNaN(u) || u <= 0 {
			continue
		}
		for range uint64(u) {
			if !rows.Next() {
				break Loop
			}
		}
		err := rows.Scan(&s)
		if err != nil {
			return "", fmt.Errorf("couldn't scan string for sample: %w", err)
		}
		w *= rand.Float64()
	}
	if rows.Err() != nil {
		return "", fmt.Errorf("couldn't get sample: %w", rows.Err())
	}
	return s, nil
}

// New creates a new prompt.
func (br *Brain) New(ctx context.Context, tag string) ([]string, error) {
	rows, err := br.stmts.newTuple.Query(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("couldn't run query for new chain: %w", err)
	}
	s, err := gumbelscan(rows)
	if err != nil {
		return nil, fmt.Errorf("couldn't get new chain: %w", err)
	}
	r := make([]string, br.order)
	r[br.order-1] = s
	return r, nil
}

// Speak creates a message from a prompt.
func (br *Brain) Speak(ctx context.Context, tag string, prompt []string) ([]string, error) {
	names := make([]sq.NamedArg, 1+len(prompt))
	names[0] = sql.Named("tag", tag)
	terms := make([]string, len(prompt))
	nn := 0
	for i, w := range prompt {
		nn += len(w) + 1
		terms[i] = brain.ReduceEntropy(w)
		names[i+1] = sql.Named("p"+strconv.Itoa(i), &terms[i])
	}
	p := make([]any, len(names))
	for i := range names {
		p[i] = names[i]
	}
	for nn < 500 {
		rows, err := br.stmts.selectTuple.Query(ctx, p...)
		if err != nil {
			return nil, fmt.Errorf("couldn't run query to continue chain with terms %v: %w", terms, err)
		}
		w, err := gumbelscan(rows)
		if err != nil {
			return nil, fmt.Errorf("couldn't continue chain with terms %v: %w", terms, err)
		}
		if w == "" {
			break
		}
		nn += len(w) + 1
		prompt = append(prompt, w)
		// Note that each p[i] is a named arg, and each name for prefix
		// elements aliases an element of terms. So, just updating terms is
		// sufficient to update the query parameters.
		copy(terms, terms[1:])
		terms[len(terms)-1] = w
	}
	return prompt, nil
}

//go:embed templates/tuple.new.sql
var newTuple string

//go:embed templates/tuple.select.sql
var selectTuple string
