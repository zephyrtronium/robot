package command

import (
	"context"
	"log/slog"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/message"
	"github.com/zephyrtronium/robot/privacy"
	"github.com/zephyrtronium/robot/spoken"
	"github.com/zephyrtronium/robot/syncmap"
	"github.com/zephyrtronium/robot/userhash"
)

// Robot is the bot state as is visible to commands.
type Robot struct {
	Log      *slog.Logger
	Channels *syncmap.Map[string, *channel.Channel]
	Brain    brain.Brain
	Privacy  *privacy.List
	Spoken   *spoken.History
}

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

// Func is a command function.
type Func func(ctx context.Context, robo *Robot, call *Invocation)
