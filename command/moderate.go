package command

import (
	"context"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

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

// Quiet makes the bot temporarily stop learning and speaking in the channel.
//   - dur: Duration to stop learning and speaking. Optional.
//   - until: Marker to stop "until tomrrow" if not empty. Optional.
//
// NOTE(zeph): Quiet waits for a timer which can be up to twelve hours.
func Quiet(ctx context.Context, robo *Robot, call *Invocation) {
	var dur time.Duration
	switch {
	case call.Args["dur"] == "" && call.Args["until"] == "":
		dur = 2 * time.Hour
	case call.Args["until"] != "":
		// The only "until" option right now is "tomorrow".
		dur = 12 * time.Hour
	default:
		if m := quietA.FindStringSubmatch(call.Args["dur"]); m != nil {
			switch m[1][0] {
			case 'h', 'H':
				dur = time.Hour
			default:
				dur = time.Minute
			}
			break
		}
		if m := quietN.FindStringSubmatch(call.Args["dur"]); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				// Should be impossible.
				call.Channel.Message(ctx, message.Format("", `sorry? (%v)`, err).AsReply(call.Message.ID))
				return
			}
			switch m[2][0] {
			case 'h', 'H':
				dur = time.Hour * time.Duration(n)
			default:
				dur = time.Minute * time.Duration(n)
			}
			break
		}
		var err error
		dur, err = time.ParseDuration(call.Args["dur"])
		if err != nil {
			call.Channel.Message(ctx, message.Format("", `sorry? (%v)`, err).AsReply(call.Message.ID))
			return
		}
	}
	if dur > 12*time.Hour {
		dur = 12 * time.Hour
	}
	call.Channel.Silent.Store(call.Message.Time().Add(dur).UnixNano())
	robo.Log.InfoContext(ctx, "silent", slog.Duration("duration", dur), slog.Time("until", call.Channel.SilentTime()))
	call.Channel.Message(ctx, message.Format("", `I won't talk or learn for %v. Some commands relating to moderation and privacy will still make me talk. I'll mention when quiet time is up.`, dur).AsReply(call.Message.ID))
	t := time.NewTimer(dur)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return
	case <-t.C:
		call.Channel.Message(ctx, message.Format("", `@%s My quiet time has ended.`, call.Message.Sender.Name))
	}
}

var (
	quietA = regexp.MustCompile(`(?i)^an?\s+(ho?u?r|mi?n)`)
	quietN = regexp.MustCompile(`(?i)^(\d+)\s+(ho?u?r|mi?n)`)
)
