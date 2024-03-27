package command

import (
	"log/slog"

	"github.com/zephyrtronium/robot/brain/kvbrain"
	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/privacy"
)

// Robot is the bot state as is visible to commands.
type Robot struct {
	Log      *slog.Logger
	Channels map[string]*channel.Channel // TODO(zeph): syncmap[string]channel.Channel
	Brain    *kvbrain.Brain
	Privacy  *privacy.List
}
