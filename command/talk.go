package command

import (
	"context"
	"log/slog"
	"math/rand/v2"

	"github.com/zephyrtronium/robot/brain"
)

func speakCmd(ctx context.Context, robo *Robot, call *Invocation) string {
	t := call.Message.Time()
	r := call.Channel.Rate.ReserveN(call.Message.Time(), 1)
	cancel := func() { r.CancelAt(t) }
	if d := r.DelayFrom(t); d > 0 {
		robo.Log.InfoContext(ctx, "won't speak; rate limited", slog.String("delay", d.String()))
		cancel()
		return ""
	}
	m, err := brain.Speak(ctx, robo.Brain, call.Channel.Send, call.Args["prompt"])
	if err != nil {
		robo.Log.ErrorContext(ctx, "couldn't speak", "err", err.Error())
		cancel()
		return ""
	}
	if call.Channel.Block.MatchString(m) {
		robo.Log.WarnContext(ctx, "generated blocked message",
			slog.String("in", call.Channel.Name),
			slog.String("text", m),
		)
		cancel()
		return ""
	}
	e := call.Channel.Emotes.Pick(rand.Uint32())
	slog.InfoContext(ctx, "speak", "in", call.Channel.Name, "text", m, "emote", e)
	return m + " " + e
}

func Speak(ctx context.Context, robo *Robot, call *Invocation) {
	u := speakCmd(ctx, robo, call)
	if u == "" {
		return
	}
	if len(u) > 450 {
		r := []rune(u)
		r = r[:min(450, len(r))]
		u = string(r)
	}
	call.Channel.Message(ctx, "", u)
}

var _ Func = Speak
