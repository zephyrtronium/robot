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
	// Recall reads out messages the brain knows.
	// At minimum, the message ID and text of each message must be retrieved;
	// other fields may be filled if they are available.
	// It must be safe to call Recall concurrently with other methods of the
	// implementation.
	// Repeated calls using the pagination token returned from the previous
	// must yield every message that the brain had recorded at the time of the
	// first call exactly once. Messages learned after the first call of an
	// enumeration are read at most once.
	// The first call of an enumeration uses an empty pagination token.
	// If the returned pagination token is empty, it is interpreted as the end
	// of the enumeration.
	Recall(ctx context.Context, tag, page string, out []Message) (n int, next string, err error)
}

var tuplesPool tpool.Pool[[]Tuple]

// Learn records a message into a Learner.
func Learn(ctx context.Context, l Learner, tag string, msg *Message) error {
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
func Recall(ctx context.Context, br Learner, tag string) iter.Seq2[Message, error] {
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
