package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"gitlab.com/zephyrtronium/tmi"
	"golang.org/x/sync/errgroup"

	"github.com/zephyrtronium/robot/v2/brain/sqlbrain"
	"github.com/zephyrtronium/robot/v2/channel"
	"github.com/zephyrtronium/robot/v2/privacy"
)

// Robot is the overall configuration for the bot.
type Robot struct {
	// brain is the brain.
	brain *sqlbrain.Brain
	// privacy is the privacy.
	privacy *privacy.DBList
	// channels are the channels.
	channels map[string]*channel.Channel
	// secrets are the bot's keys.
	secrets *keys
	// owner is the username of the owner.
	owner string
	// ownerContact describes contact information for the owner.
	ownerContact string
	// http is the bot's HTTP server configuration.
	http http.Server
	// tmi contains the bot's Twitch OAuth2 settings. It may be nil if there is
	// no Twitch configuration.
	tmi *client
}

func (robo *Robot) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)
	// TODO(zeph): stdin?
	// TODO(zeph): don't like having this here
	robo.http.BaseContext = func(l net.Listener) context.Context { return ctx }
	// TODO(zeph): tls
	go robo.http.ListenAndServe()
	if robo.tmi != nil {
		group.Go(func() error { return robo.twitch(ctx, group) })
	}
	err := group.Wait()
	if err == context.Canceled {
		// If the first error is context canceled, then we are shutting down
		// normally in response to a sigint.
		err = nil
	}
	return err
}

func (robo *Robot) twitch(ctx context.Context, group *errgroup.Group) error {
	tok, err := robo.tmi.token.Access(ctx)
	if err != nil {
		return fmt.Errorf("couldn't obtain access token for TMI login: %w", err)
	}
	// TODO(zeph): resolve usernames in configs to user ids
	cfg := tmi.ConnectConfig{
		Dial:         new(tls.Dialer).DialContext,
		RetryWait:    tmi.RetryList(true, 0, time.Second, time.Minute, 5*time.Minute),
		Nick:         strings.ToLower(robo.tmi.me),
		Pass:         "oauth:" + tok,
		Capabilities: []string{"twitch.tv/commands", "twitch.tv/tags"},
		Timeout:      300 * time.Second,
	}
	send := make(chan *tmi.Message, 1)
	recv := make(chan *tmi.Message, 8) // 8 is enough for on-connect msgs
	// TODO(zeph): could run several instances of loop
	go robo.loop(ctx, send, recv)
	tmi.Connect(ctx, cfg, tmi.Log(log.Default(), false), send, recv)
	return ctx.Err()
}

func (robo *Robot) loop(ctx context.Context, send chan<- *tmi.Message, recv <-chan *tmi.Message) {
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
				// TODO(zeph): this
			case "WHISPER":
				// TODO(zeph): this
			case "NOTICE":
				// nothing yet
			case "CLEARCHAT":
				// TODO(zeph): forget messages from the target (even self)
			case "CLEARMSG":
				// TODO(zeph): forget message
			case "HOSTTARGET":
				// nothing yet
			case "USERSTATE":
				// We used to check our badges and update our hard rate limit
				// per-channel, but per-channel rate limits only really make
				// sense for verified bots which have a relaxed global limit.
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
		l := ls
		if len(l) > burst {
			l = l[:burst]
		}
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
