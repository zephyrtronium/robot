package command

import (
	"log/slog"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/privacy"
	"github.com/zephyrtronium/robot/spoken"
)

// Robot is the bot state as is visible to commands.
type Robot struct {
	Log      *slog.Logger
	Channels map[string]*channel.Channel // TODO(zeph): syncmap[string]channel.Channel
	Brain    brain.Brain
	Privacy  *privacy.List
	Spoken   *spoken.History
}
