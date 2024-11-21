package command

import (
	"context"
	"log/slog"
	"math/rand/v2"

	"github.com/zephyrtronium/robot/message"
)

func Private(ctx context.Context, robo *Robot, call *Invocation) {
	err := robo.Privacy.Add(ctx, call.Message.Sender)
	if err != nil {
		robo.Log.ErrorContext(ctx, "privacy add failed", slog.Any("err", err), slog.String("channel", call.Channel.Name))
		call.Channel.Message(ctx, message.Format("", "Something went wrong while trying to add you to the privacy list. Try again. Sorry!").AsReply(call.Message.ID))
		return
	}
	e := call.Channel.Emotes.Pick(rand.Uint32())
	call.Channel.Message(ctx, message.Format("", `Sure, I won't learn from your messages. Most of my functionality will still work for you. If you'd like to have me learn from you again, just tell me, "learn from me again." %s`, e).AsReply(call.Message.ID))
}

func Unprivate(ctx context.Context, robo *Robot, call *Invocation) {
	err := robo.Privacy.Remove(ctx, call.Message.Sender)
	if err != nil {
		robo.Log.ErrorContext(ctx, "privacy remove failed", slog.Any("err", err), slog.String("channel", call.Channel.Name))
		call.Channel.Message(ctx, message.Format("", "Something went wrong while trying to add you to the privacy list. Try again. Sorry!").AsReply(call.Message.ID))
		return
	}
	e := call.Channel.Emotes.Pick(rand.Uint32())
	call.Channel.Message(ctx, message.Format("", `Sure, I'll learn from you again! %s`, e).AsReply(call.Message.ID))
}

func DescribePrivacy(ctx context.Context, robo *Robot, call *Invocation) {
	// TODO(zeph): describe privacy
	call.Channel.Message(ctx, message.Format("", `See here for a description of what information I collect, and how to opt out of all collection: https://github.com/zephyrtronium/robot#what-data-does-robot-store`).AsReply(call.Message.ID))
}
