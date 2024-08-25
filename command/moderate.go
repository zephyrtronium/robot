package command

import (
	"context"
	"log/slog"
	"strings"
)

func Forget(ctx context.Context, robo *Robot, call *Invocation) {
	h := call.Channel.History.Messages()
	term := strings.ToLower(call.Args["term"])
	for _, m := range h {
		if !strings.Contains(strings.ToLower(m.Text), term) {
			continue
		}
		robo.Log.DebugContext(ctx, "forget",
			slog.String("tag", call.Channel.Learn),
			slog.String("id", m.ID),
		)
		err := robo.Brain.ForgetMessage(ctx, call.Channel.Learn, m.ID)
		if err != nil {
			robo.Log.ErrorContext(ctx, "failed to forget",
				slog.Any("err", err),
				slog.String("tag", call.Channel.Learn),
				slog.String("id", m.ID),
			)
		}
	}
}
