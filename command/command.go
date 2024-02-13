package command

import (
	"context"

	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/message"
	"github.com/zephyrtronium/robot/userhash"
)

// CommandFunc executes a command.
type CommandFunc func(ctx context.Context, robo *Robot, hasher userhash.Hasher, ch *channel.Channel, send func(message.Interface))
