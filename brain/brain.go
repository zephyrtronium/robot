package brain

import (
	"github.com/zephyrtronium/robot/message"
	"github.com/zephyrtronium/robot/userhash"
)

// Brain is a combined [Learner] and [Speaker].
type Brain interface {
	Learner
	Speaker
}

// Message is the message type used by a [Brain].
type Message = message.Received[userhash.Hash]
