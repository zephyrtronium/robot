package command

import (
	"context"
	"log/slog"
	"strings"

	"github.com/zephyrtronium/robot/message"
)

func Forget(ctx context.Context, robo *Robot, call *Invocation) {
	h := call.Channel.History.All()
	term := strings.ToLower(call.Args["term"])
	n := 0
	for m := range h {
		if !strings.Contains(strings.ToLower(m.Text), term) {
			continue
		}
		n++
		robo.Log.DebugContext(ctx, "forget",
			slog.String("tag", call.Channel.Learn),
			slog.String("id", m.ID),
		)
		robo.Metrics.ForgotCount.Observe(1)
		err := robo.Brain.Forget(ctx, call.Channel.Learn, m.ID)
		if err != nil {
			robo.Log.ErrorContext(ctx, "failed to forget",
				slog.Any("err", err),
				slog.String("tag", call.Channel.Learn),
				slog.String("id", m.ID),
			)
		}
	}
	switch n {
	case 0:
		call.Channel.Message(ctx, message.Format("", "No messages contained %q.", term).AsReply(call.Message.ID))
	case 1:
		call.Channel.Message(ctx, message.Format("", "Forgot 1 message."))
	default:
		call.Channel.Message(ctx, message.Format("", "Forgot %d messages.", n).AsReply(call.Message.ID))
	}
}
