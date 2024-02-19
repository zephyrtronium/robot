package command

import (
	"context"

	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/message"
	"github.com/zephyrtronium/robot/userhash"
)

// Invocation is a command invocation. An Invocation and its fields must not
// be modified or retained by any command.
type Invocation struct {
	// Channel is the channel where the invocation occurred.
	Channel *channel.Channel
	// Message is the message which triggered the invocation. It is always
	// non-nil, but not all fields are guaranteed to be populated.
	Message *message.Received
	// Args is the parsed arguments to the command.
	Args map[string]string
	// Hasher is a user hasher for the command's use.
	Hasher userhash.Hasher
}

// Func executes a command.
type Func func(ctx context.Context, robo *Robot, call *Invocation)
