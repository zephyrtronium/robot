package brain

import (
	"context"
	"iter"

	"github.com/zephyrtronium/robot/message"
	"github.com/zephyrtronium/robot/userhash"
)

// Interface is a store of learned messages which can reproduce them by parts.
//
// It must be safe to call all methods of a brain concurrently with each other.
type Interface interface {
	// Learn records a set of tuples.
	//
	// One tuple has an empty prefix to denote the start of the message, and
	// a different tuple has the empty string as its suffix to denote the end
	// of the message. The positions of each in the argument are not guaranteed.
	//
	// Tuples in the argument may share storage for prefixes.
	Learn(ctx context.Context, tag string, msg *Message, tuples []Tuple) error

	// Think iterates all suffixes matching a prefix.
	//
	// Yielded closures replace id and suffix with successive messages' contents.
	// The iterating loop may not call the yielded closure on every iteration.
	// When it does, it uses the closure fully within each loop, allowing the
	// sequence to yield the same closure on each call.
	// Conversely, iterators should expect that the arguments to the closure
	// will be the same on each iteration and must not retain them.
	Think(ctx context.Context, tag string, prefix []string) iter.Seq[func(id, suf *[]byte) error]

	// Forget forgets everything learned from a single given message.
	// If nothing has been learned from the message, it must prevent anything
	// from being learned from a message with that ID.
	Forget(ctx context.Context, tag, id string) error

	// Recall reads out messages the brain knows.
	// At minimum, the message ID and text of each message must be retrieved;
	// other fields may be filled if they are available.
	//
	// Repeated calls using the pagination token returned from the previous
	// must yield every message that the brain had recorded at the time of the
	// first call exactly once. Messages learned after the first call of an
	// enumeration are read at most once.
	//
	// The first call of an enumeration uses an empty pagination token as input.
	// If the returned pagination token is empty, it is interpreted as the end
	// of the enumeration.
	Recall(ctx context.Context, tag, page string, out []Message) (n int, next string, err error)
}

// Message is the message type used by a [Interface].
type Message = message.Received[userhash.Hash]
