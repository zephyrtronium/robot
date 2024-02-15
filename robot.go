package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"gitlab.com/zephyrtronium/tmi"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/zephyrtronium/robot/auth"
	"github.com/zephyrtronium/robot/brain/sqlbrain"
	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/privacy"
)

// Robot is the overall configuration for the bot.
type Robot struct {
	// brain is the brain.
	brain *sqlbrain.Brain
	// privacy is the privacy.
	privacy *privacy.List
	// channels are the channels.
	channels map[string]*channel.Channel // TODO(zeph): syncmap[string]channel.Channel
	// works is the worker queue.
	works chan chan func(context.Context)
	// secrets are the bot's keys.
	secrets *keys
	// owner is the username of the owner.
	owner string
	// ownerContact describes contact information for the owner.
	ownerContact string
	// tmi contains the bot's Twitch OAuth2 settings. It may be nil if there is
	// no Twitch configuration.
	tmi *client
}

// client is the settings for OAuth2 and related elements.
type client struct {
	// me is the bot's username. The interpretation of this is domain-specific.
	me string
	// owner is the user ID of the owner. The interpretation of this is
	// domain-specific.
	owner string
	// rate is the global rate limiter for this client.
	rate *rate.Limiter
	// token is the OAuth2 token.
	token *auth.Token
}

// New creates a new robot instance. Use SetOwner, SetSecrets, &c. as needed
// to initialize the robot.
func New(poolSize int) *Robot {
	return &Robot{
		channels: make(map[string]*channel.Channel),
		works:    make(chan chan func(context.Context), poolSize),
	}
}

func (robo *Robot) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)
	// TODO(zeph): stdin?
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
	go robo.tmiLoop(ctx, send, recv)
	tmi.Connect(ctx, cfg, tmi.Log(log.Default(), false), send, recv)
	return ctx.Err()
}

func (robo *Robot) tmiLoop(ctx context.Context, send chan<- *tmi.Message, recv <-chan *tmi.Message) {
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
				robo.tmiMessage(ctx, send, msg)
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
