package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"gitlab.com/zephyrtronium/tmi"
	"golang.org/x/sync/errgroup"

	"github.com/zephyrtronium/robot/userhash"
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
				robo.tmiMessage(ctx, group, send, msg)
			case "WHISPER":
				// TODO(zeph): this
			case "NOTICE":
				// nothing yet
			case "CLEARCHAT":
				robo.clearchat(ctx, group, msg)
			case "CLEARMSG":
				robo.clearmsg(ctx, group, msg)
			case "HOSTTARGET":
				// nothing yet
			case "USERSTATE":
				// We used to check our badges and update our hard rate limit
				// per-channel, but per-channel rate limits only really make
				// sense for verified bots which have a relaxed global limit.
			case "GLOBALUSERSTATE":
				slog.InfoContext(ctx, "connected to TMI", slog.String("GLOBALUSERSTATE", msg.Tags))
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

func (robo *Robot) clearchat(ctx context.Context, group *errgroup.Group, msg *tmi.Message) {
	if len(msg.Params) == 0 {
		return
	}
	ch, _ := robo.channels.Load(msg.To())
	if ch == nil {
		return
	}
	var work func(ctx context.Context)
	t, _ := msg.Tag("target-user-id")
	switch t {
	case "":
		// Delete all recent chat.
		work = func(ctx context.Context) {
			tag := ch.Learn
			slog.InfoContext(ctx, "clear all chat", slog.String("channel", msg.To()), slog.String("tag", tag))
			err := robo.brain.ForgetDuring(ctx, tag, msg.Time().Add(-15*time.Minute), msg.Time())
			if err != nil {
				slog.ErrorContext(ctx, "failed to forget from all chat", slog.Any("err", err), slog.String("channel", msg.To()))
			}
		}
	case robo.tmi.userID:
		work = func(ctx context.Context) {
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
				if err := robo.brain.ForgetMessage(ctx, tag, id); err != nil {
					slog.ErrorContext(ctx, "failed to forget from recent trace",
						slog.Any("err", err),
						slog.String("channel", msg.To()),
						slog.String("tag", tag),
						slog.String("id", id),
					)
				}
			}
		}
	default:
		// Delete from user.
		// We use the user's current and previous userhash, since userhashes
		// are time-based.
		work = func(ctx context.Context) {
			hr := userhash.New(robo.secrets.userhash)
			h := hr.Hash(new(userhash.Hash), t, msg.To(), msg.Time())
			if err := robo.brain.ForgetUser(ctx, h); err != nil {
				slog.ErrorContext(ctx, "failed to forget recent messages from user", slog.Any("err", err), slog.String("channel", msg.To()))
				// Try the previous userhash anyway.
			}
			h = hr.Hash(h, t, msg.To(), msg.Time().Add(-userhash.TimeQuantum))
			if err := robo.brain.ForgetUser(ctx, h); err != nil {
				slog.ErrorContext(ctx, "failed to forget older messages from user", slog.Any("err", err), slog.String("channel", msg.To()))
			}
		}
	}
	robo.enqueue(ctx, group, work)
}

func (robo *Robot) clearmsg(ctx context.Context, group *errgroup.Group, msg *tmi.Message) {
	if len(msg.Params) == 0 {
		return
	}
	ch, _ := robo.channels.Load(msg.To())
	if ch == nil {
		return
	}
	t, _ := msg.Tag("target-msg-id")
	u, _ := msg.Tag("login")
	work := func(ctx context.Context) {
		if u != robo.tmi.name {
			// Forget a message from someone else.
			slog.InfoContext(ctx, "forget message", slog.String("channel", msg.To()), slog.String("id", t))
			err := robo.brain.ForgetMessage(ctx, ch.Learn, t)
			if err != nil {
				slog.ErrorContext(ctx, "failed to forget message",
					slog.Any("err", err),
					slog.String("channel", msg.To()),
					slog.String("tag", ch.Learn),
					slog.String("id", t),
				)
			}
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
			slog.ErrorContext(ctx, "failed to get message trace",
				slog.Any("err", err),
				slog.String("channel", msg.To()),
				slog.String("tag", ch.Send),
				slog.String("id", t),
			)
			return
		}
		slog.InfoContext(ctx, "forget trace",
			slog.String("channel", msg.To()),
			slog.String("tag", ch.Send),
			slog.Any("learned", tm),
			slog.Any("trace", trace),
		)
		for _, id := range trace {
			err := robo.brain.ForgetMessage(ctx, ch.Send, id)
			if err != nil {
				slog.ErrorContext(ctx, "failed to forget from trace",
					slog.Any("err", err),
					slog.String("channel", msg.To()),
					slog.String("tag", ch.Send),
					slog.String("id", t),
				)
			}
		}
	}
	robo.enqueue(ctx, group, work)
}
