package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"gitlab.com/zephyrtronium/tmi"
	"golang.org/x/sync/errgroup"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/metrics"
)

func (robo *Robot) tmiLoop(ctx context.Context, group *errgroup.Group, send chan<- *tmi.Message, recv <-chan *tmi.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-recv:
			if !ok {
				return
			}
			switch msg.Command {
			case "PRIVMSG":
				group.Go(func() error {
					robo.tmiMessage(ctx, send, msg)
					return nil
				})
			case "WHISPER":
				// TODO(zeph): this
			case "NOTICE":
				// nothing yet
			case "CLEARCHAT":
				group.Go(func() error {
					robo.clearchat(ctx, msg)
					return nil
				})
			case "CLEARMSG":
				group.Go(func() error {
					robo.clearmsg(ctx, msg)
					return nil
				})
			case "HOSTTARGET":
				// nothing yet
			case "USERSTATE":
				// We used to check our badges and update our hard rate limit
				// per-channel, but per-channel rate limits only really make
				// sense for verified bots which have a relaxed global limit.
			case "GLOBALUSERSTATE":
				slog.InfoContext(ctx, "connected to TMI", slog.String("GLOBALUSERSTATE", msg.Tags))
			case "366": // End NAMES
				if len(msg.Params) > 1 {
					slog.InfoContext(ctx, "joined channel", slog.String("channel", msg.Params[1]))
				}
			case "376": // End MOTD
				go robo.joinTwitch(ctx, send)
			}
		}
	}
}

func (robo *Robot) joinTwitch(ctx context.Context, send chan<- *tmi.Message) {
	ls := make([]string, 0, robo.channels.Len())
	for _, ch := range robo.channels.All() {
		ls = append(ls, ch.Name)
	}
	burst := 20
	for len(ls) > 0 {
		l := ls[:min(burst, len(ls))]
		ls = ls[len(l):]
		msg := tmi.Message{
			Command: "JOIN",
			Params:  []string{strings.Join(l, ",")},
		}
		select {
		case <-ctx.Done():
			return
		case send <- &msg:
			// do nothing
		}
		if len(ls) > 0 {
			// Per https://dev.twitch.tv/docs/irc/#rate-limits we get 20 join
			// attempts per ten seconds. Use a slightly longer delay to ensure
			// we don't get globaled by clock drift.
			time.Sleep(11 * time.Second)
		}
	}
}

func (robo *Robot) clearchat(ctx context.Context, msg *tmi.Message) {
	if len(msg.Params) == 0 {
		return
	}
	ch, _ := robo.channels.Load(msg.To())
	if ch == nil {
		return
	}
	t, _ := msg.Tag("target-user-id")
	switch t {
	case "":
		// Delete all recent chat.
		tag := ch.Learn
		slog.InfoContext(ctx, "clear all chat", slog.String("channel", msg.To()), slog.String("tag", tag))
		for m := range ch.History.All() {
			slog.DebugContext(ctx, "forget all chat", slog.String("channel", msg.To()), slog.String("id", m.ID))
			robo.metrics.ForgotCount.Observe(1)
			err := robo.brain.Forget(ctx, tag, m.ID)
			if err != nil {
				slog.ErrorContext(ctx, "failed to forget while clearing all chat",
					slog.Any("err", err),
					slog.String("channel", msg.To()),
					slog.String("id", m.ID),
				)
			}
		}
	case robo.tmi.userID:
		// We use the send tag because we are forgetting something we sent.
		tag := ch.Send
		slog.InfoContext(ctx, "forget recent generated", slog.String("channel", msg.To()), slog.String("tag", tag))
		for id, err := range robo.spoken.Since(ctx, tag, msg.Time().Add(-15*time.Minute)) {
			if err != nil {
				slog.ErrorContext(ctx, "failed to get recent traces",
					slog.Any("err", err),
					slog.String("channel", msg.To()),
					slog.String("tag", tag),
				)
				continue
			}
			slog.DebugContext(ctx, "forget from recent trace", slog.String("channel", msg.To()), slog.String("id", id))
			robo.metrics.ForgotCount.Observe(1)
			if err := robo.brain.Forget(ctx, tag, id); err != nil {
				slog.ErrorContext(ctx, "failed to forget from recent trace",
					slog.Any("err", err),
					slog.String("channel", msg.To()),
					slog.String("tag", tag),
					slog.String("id", id),
				)
			}
		}
	default:
		// Delete from user.
		for m := range ch.History.All() {
			if m.Sender.ID != t {
				continue
			}
			slog.DebugContext(ctx, "forget from user", slog.String("channel", msg.To()), slog.String("id", m.ID))
			robo.metrics.ForgotCount.Observe(1)
			if err := robo.brain.Forget(ctx, ch.Learn, m.ID); err != nil {
				slog.ErrorContext(ctx, "failed to forget from user",
					slog.Any("err", err),
					slog.String("channel", msg.To()),
					slog.String("id", m.ID),
				)
			}
		}
	}
}

func (robo *Robot) clearmsg(ctx context.Context, msg *tmi.Message) {
	if len(msg.Params) == 0 {
		return
	}
	ch, _ := robo.channels.Load(msg.To())
	if ch == nil {
		return
	}
	t, _ := msg.Tag("target-msg-id")
	u, _ := msg.Tag("login")
	log := slog.With(slog.String("trace", t), slog.String("in", msg.To()))
	if u != robo.tmi.name {
		// Forget a message from someone else.
		log.InfoContext(ctx, "forget message", slog.String("tag", ch.Learn), slog.String("id", t))
		forget(ctx, log, robo.metrics.ForgotCount, robo.brain, ch.Learn, t)
		return
	}
	// Forget a message from the robo.
	// This may or may not be a generated message; it could be a command
	// output or copypasta. Regardless, if it was deleted, we should try
	// not to say it.
	// Note that we use the send tag rather than the learn tag for this,
	// because we are unlearning something that we sent.
	trace, tm, err := robo.spoken.Trace(ctx, ch.Send, msg.Trailing)
	if err != nil {
		log.ErrorContext(ctx, "failed to get message trace",
			slog.Any("err", err),
			slog.String("tag", ch.Send),
			slog.String("text", msg.Trailing),
			slog.String("id", t),
		)
		return
	}
	log.InfoContext(ctx, "forget trace", slog.String("tag", ch.Send), slog.Any("spoken", tm), slog.Any("trace", trace))
	forget(ctx, log, robo.metrics.ForgotCount, robo.brain, ch.Send, trace...)
}

func forget(ctx context.Context, log *slog.Logger, forgetCount metrics.Observer, brain brain.Interface, tag string, trace ...string) {
	forgetCount.Observe(1)
	for _, id := range trace {
		err := brain.Forget(ctx, tag, id)
		if err != nil {
			log.ErrorContext(ctx, "failed to forget message",
				slog.Any("err", err),
				slog.String("tag", tag),
				slog.String("id", id),
			)
		}
	}
}
