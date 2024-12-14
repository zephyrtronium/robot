package brain

import (
	"context"
	"iter"
	"slices"

	"github.com/zephyrtronium/robot/tpool"
)

// Tuple is a single Markov chain tuple.
type Tuple struct {
	// Prefix is the entropy-reduced prefix in reverse order relative to the
	// source message.
	Prefix []string
	// Suffix is the full-entropy term following the prefix.
	Suffix string
}

var tuplesPool tpool.Pool[[]Tuple]

// Learn records a message into a brain.
func Learn(ctx context.Context, l Interface, tag string, msg *Message) error {
	toks := tokens(tokensPool.Get(), msg.Text)
	defer func() { tokensPool.Put(toks[:0]) }()
	if len(toks) == 0 {
		return nil
	}
	tt := tuplesPool.Get()
	defer func() { tuplesPool.Put(tt[:0]) }()
	tt = slices.Grow(tt, len(toks)+1)
	tt = tupleToks(tt, toks)
	return l.Learn(ctx, tag, msg, tt)
}

func tupleToks(tt []Tuple, toks []string) []Tuple {
	slices.Reverse(toks)
	pres := slices.Clone(toks)
	for i, w := range pres {
		pres[i] = ReduceEntropy(w)
	}
	suf := ""
	for i, w := range toks {
		tt = append(tt, Tuple{Prefix: pres[i:], Suffix: suf})
		suf = w
	}
	tt = append(tt, Tuple{Prefix: nil, Suffix: suf})
	return tt
}

// Recall iterates over all messages a brain knows with a given tag.
func Recall(ctx context.Context, br Interface, tag string) iter.Seq2[Message, error] {
	return func(yield func(Message, error) bool) {
		var (
			page string
			n    int
			err  error
		)
		msgs := make([]Message, 64)
		for {
			n, page, err = br.Recall(ctx, tag, page, msgs)
			if err != nil {
				yield(Message{}, err)
				return
			}
			for i := range msgs[:n] {
				if !yield(msgs[i], nil) {
					return
				}
			}
			if page == "" {
				return
			}
		}
	}
}
