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
				// TODO(zeph): forget message
			case "HOSTTARGET":
				// nothing yet
			case "USERSTATE":
				// We used to check our badges and update our hard rate limit
				// per-channel, but per-channel rate limits only really make
				// sense for verified bots which have a relaxed global limit.
			case "GLOBALUSERSTATE":
				slog.InfoContext(ctx, "connected to TMI", slog.String("GLOBALUSERSTATE", msg.Tags))
			case "376": // End MOTD
				go robo.join(ctx, send)
			}
		}
	}
}

func (robo *Robot) join(ctx context.Context, send chan<- *tmi.Message) {
	ls := make([]string, 0, len(robo.channels))
	// TODO(zeph): this is going to join channels that aren't twitch
	for _, ch := range robo.channels {
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
	ch := robo.channels[msg.To()]
	if ch == nil {
		return
	}
	var work func(ctx context.Context)
	t, _ := msg.Tag("target-user-id")
	switch {
	case t == "":
		// Delete all recent chat.
		work = func(ctx context.Context) {
			tag := ch.Learn
			slog.InfoContext(ctx, "clear all chat", slog.String("channel", msg.To()), slog.String("tag", tag))
			err := robo.brain.ForgetDuring(ctx, tag, msg.Time().Add(-15*time.Minute), msg.Time())
			if err != nil {
				slog.ErrorContext(ctx, "failed to forget from all chat", slog.Any("err", err), slog.String("channel", msg.To()))
			}
		}
	case msg.Trailing == robo.tmi.me: // TODO(zeph): get own user id
		// TODO(zeph): forget all recent generated traces
		return
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
