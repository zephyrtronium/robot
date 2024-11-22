package brain

import (
	"context"
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

// Learner records Markov chain tuples.
type Learner interface {
	// Learn records a set of tuples.
	// One tuple has an empty prefix to denote the start of the message, and
	// a different tuple has the empty string as its suffix to denote the end
	// of the message. The positions of each in the argument are not guaranteed.
	// Each tuple's prefix has entropy reduction transformations applied.
	// Tuples in the argument may share storage for prefixes.
	Learn(ctx context.Context, tag string, msg *Message, tuples []Tuple) error
	// Forget forgets everything learned from a single given message.
	// If nothing has been learned from the message, it should prevent anything
	// from being learned from a message with that ID.
	Forget(ctx context.Context, tag, id string) error
}

var tuplesPool tpool.Pool[[]Tuple]

// Learn records a message into a Learner.
func Learn(ctx context.Context, l Learner, tag string, msg *Message) error {
	toks := Tokens(tokensPool.Get(), msg.Text)
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
