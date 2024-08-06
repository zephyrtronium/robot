package brain

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/zephyrtronium/robot/tpool"
	"github.com/zephyrtronium/robot/userhash"
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
	Learn(ctx context.Context, tag string, user userhash.Hash, id uuid.UUID, t time.Time, tuples []Tuple) error
	// Forget removes a set of recorded tuples.
	// The tuples provided are as for Learn.
	// If a tuple has been recorded multiple times, only the first
	// should be deleted.
	// If a tuple has not been recorded, it should be ignored.
	Forget(ctx context.Context, tag string, tuples []Tuple) error
	// ForgetMessage forgets everything learned from a single given message.
	// If nothing has been learned from the message, it should be ignored.
	ForgetMessage(ctx context.Context, tag string, msg uuid.UUID) error
	// ForgetDuring forgets all messages learned in the given time span.
	ForgetDuring(ctx context.Context, tag string, since, before time.Time) error
	// ForgetUser forgets all messages associated with a userhash.
	ForgetUser(ctx context.Context, user *userhash.Hash) error
}

var tuplesPool tpool.Pool[[]Tuple]

// Learn records tokens into a Learner.
func Learn(ctx context.Context, l Learner, tag string, user userhash.Hash, id uuid.UUID, t time.Time, toks []string) error {
	if len(toks) == 0 {
		return nil
	}
	tt := tuplesPool.Get()
	defer func() { tuplesPool.Put(tt[:0]) }()
	tt = slices.Grow(tt, len(toks)+1)
	tt = tupleToks(tt, toks)
	return l.Learn(ctx, tag, user, id, t, tt)
}

// Forget removes tokens from a Learner.
func Forget(ctx context.Context, l Learner, tag string, toks []string) error {
	if len(toks) == 0 {
		return nil
	}
	tt := tuplesPool.Get()
	defer func() { tuplesPool.Put(tt[:0]) }()
	tt = slices.Grow(tt, len(toks)+1)
	tt = tupleToks(tt, toks)
	return l.Forget(ctx, tag, tt)
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
