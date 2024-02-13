package command

import (
	"log/slog"

	"github.com/zephyrtronium/robot/brain/sqlbrain"
	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/privacy"
)

// Robot is the bot state as is visible to commands.
type Robot struct {
	Log      *slog.Logger
	Channels map[string]*channel.Channel // TODO(zeph): not pointer?
	Brain    *sqlbrain.Brain
	Privacy  *privacy.List
}
