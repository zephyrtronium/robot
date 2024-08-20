package command

import (
	"context"
	"log/slog"
)

// EchoIn sends a plain text message to any channel.
//   - in: Name of the channel to send to.
//   - msg: Message to send.
func EchoIn(ctx context.Context, robo *Robot, call *Invocation) {
	t := call.Args["in"]
	ch := robo.Channels[t]
	if ch == nil {
		robo.Log.WarnContext(ctx, "echo into unknown channel", slog.String("target", t))
		return
	}
	ch.Message(ctx, "", call.Args["msg"])
}

// Echo sends a plain text message to the channel in which it is invoked.
//   - msg: Message to send.
func Echo(ctx context.Context, robo *Robot, call *Invocation) {
	call.Channel.Message(ctx, "", call.Args["msg"])
}
