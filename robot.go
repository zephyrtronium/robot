package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gitlab.com/zephyrtronium/tmi"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/zephyrtronium/robot/auth"
	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/privacy"
	"github.com/zephyrtronium/robot/spoken"
	"github.com/zephyrtronium/robot/twitch"
)

// Robot is the overall configuration for the bot.
type Robot struct {
	// brain is the brain.
	brain brain.Brain
	// privacy is the privacy.
	privacy *privacy.List
	// spoken is the history of generated messages.
	spoken *spoken.History
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
	tmi *client[*tmi.Message, *tmi.Message]
	// twitch is the Twitch API client.
	twitch twitch.Client
}

// client is the settings for OAuth2 and related elements.
// The type parameter is the type of messages sent TO the service.
type client[Send, Receive any] struct {
	// send is the channel on which messages are sent.
	send chan Send
	// recv is the channel on which received messages are communicated.
	recv chan Receive
	// id is the application client ID.
	id string
	// me is the bot's username. The interpretation of this is domain-specific.
	me string
	// owner is the user ID of the owner. The interpretation of this is
	// domain-specific.
	owner string
	// rate is the global rate limiter for this client.
	rate *rate.Limiter
	// tokens is the source of OAuth2 tokens.
	tokens auth.TokenSource
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
		group.Go(func() error { return robo.runTwitch(ctx, group) })
	}
	err := group.Wait()
	if err == context.Canceled {
		// If the first error is context canceled, then we are shutting down
		// normally in response to a sigint.
		err = nil
	}
	return err
}

func (robo *Robot) runTwitch(ctx context.Context, group *errgroup.Group) error {
	tok, err := twitchToken(ctx, robo.tmi.tokens)
	if err != nil {
		return err
	}
	cfg := tmi.ConnectConfig{
		Dial:         new(tls.Dialer).DialContext,
		RetryWait:    tmi.RetryList(true, 0, time.Second, time.Minute, 5*time.Minute),
		Nick:         strings.ToLower(robo.tmi.me),
		Pass:         "oauth:" + tok.AccessToken,
		Capabilities: []string{"twitch.tv/commands", "twitch.tv/tags"},
		Timeout:      300 * time.Second,
	}
	group.Go(func() error {
		robo.tmiLoop(ctx, group, robo.tmi.send, robo.tmi.recv)
		return nil
	})
	group.Go(func() error {
		return robo.streamsLoop(ctx, robo.channels)
	})
	tmi.Connect(ctx, cfg, &tmiSlog{slog.Default()}, robo.tmi.send, robo.tmi.recv)
	return ctx.Err()
}

// twitchToken gets a valid Twitch access token by refreshing until it
// validates successfully.
func twitchToken(ctx context.Context, tokens auth.TokenSource) (*oauth2.Token, error) {
	tok, err := tokens.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't obtain Twitch access token: %w", err)
	}
	client := http.Client{Timeout: 30 * time.Second}
	for range 5 {
		v, err := twitch.Validate(ctx, &client, tok)
		slog.InfoContext(ctx, "Twitch validation", slog.Any("response", v), slog.Any("err", err))
		switch {
		case err == nil:
			// Current token is good.
			return tok, nil
		case errors.Is(err, twitch.ErrNeedRefresh):
			// Refresh and try again.
			tok, err = tokens.Refresh(ctx, tok)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("couldn't refresh Twitch access token: %w", err)
		}
	}
	return nil, fmt.Errorf("giving up on refresh retries")
}

func (robo *Robot) streamsLoop(ctx context.Context, channels map[string]*channel.Channel) error {
	// TODO(zeph): one day we should switch to eventsub
	// TODO(zeph): remove anything learned since the last check when offline
	tok, err := robo.tmi.tokens.Token(ctx)
	if err != nil {
		return err
	}
	streams := make([]twitch.Stream, 0, len(channels))
	m := make(map[string]bool, len(channels))
	// Run once at the start so we start learning in online streams immediately.
	streams = streams[:0]
	for _, ch := range channels {
		n := strings.ToLower(strings.TrimPrefix(ch.Name, "#"))
		streams = append(streams, twitch.Stream{UserLogin: n})
	}
	for range 5 {
		// TODO(zeph): limit to 100
		streams, err = twitch.UserStreams(ctx, robo.twitch, tok, streams)
		switch err {
		case nil:
			slog.InfoContext(ctx, "stream infos", slog.Int("count", len(streams)))
			// Mark online streams as enabled.
			// First map names to online status.
			for _, s := range streams {
				slog.DebugContext(ctx, "stream",
					slog.String("login", s.UserLogin),
					slog.String("display", s.UserName),
					slog.String("id", s.UserID),
					slog.String("type", s.Type),
				)
				n := strings.ToLower(s.UserLogin)
				m[n] = true
			}
			// Now loop all streams.
			for _, ch := range channels {
				n := strings.ToLower(strings.TrimPrefix(ch.Name, "#"))
				ch.Enabled.Store(m[n])
			}
		case twitch.ErrNeedRefresh:
			tok, err = twitchToken(ctx, robo.tmi.tokens)
			if err != nil {
				slog.ErrorContext(ctx, "failed to refresh token", slog.Any("err", err))
				return fmt.Errorf("couldn't get valid access token: %w", err)
			}
			continue
		default:
			slog.ErrorContext(ctx, "failed to query online broadcasters", slog.Any("streams", streams), slog.Any("err", err))
			// All streams are already offline.
		}
		break
	}
	streams = streams[:0]
	clear(m)

	tick := time.NewTicker(time.Minute)
	go func() {
		<-ctx.Done()
		tick.Stop()
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			for _, ch := range channels {
				n := strings.TrimPrefix(ch.Name, "#")
				streams = append(streams, twitch.Stream{UserLogin: n})
			}
			for range 5 {
				// TODO(zeph): limit to 100
				streams, err = twitch.UserStreams(ctx, robo.twitch, tok, streams)
				switch err {
				case nil:
					slog.InfoContext(ctx, "stream infos", slog.Int("count", len(streams)))
					// Mark online streams as enabled.
					// First map names to online status.
					for _, s := range streams {
						slog.DebugContext(ctx, "stream",
							slog.String("login", s.UserLogin),
							slog.String("display", s.UserName),
							slog.String("id", s.UserID),
							slog.String("type", s.Type),
						)
						n := strings.ToLower(s.UserLogin)
						m[n] = true
					}
					// Now loop all streams.
					for _, ch := range channels {
						n := strings.ToLower(strings.TrimPrefix(ch.Name, "#"))
						ch.Enabled.Store(m[n])
					}
				case twitch.ErrNeedRefresh:
					tok, err = twitchToken(ctx, robo.tmi.tokens)
					if err != nil {
						slog.ErrorContext(ctx, "failed to refresh token", slog.Any("err", err))
						return fmt.Errorf("couldn't get valid access token: %w", err)
					}
					continue
				default:
					slog.ErrorContext(ctx, "failed to query online broadcasters", slog.Any("streams", streams), slog.Any("err", err))
					// Set all streams as offline.
					for _, ch := range channels {
						ch.Enabled.Store(false)
					}
				}
				break
			}
			streams = streams[:0]
			clear(m)
		}
	}
}

func deviceCodePrompt(userCode, verURI, verURIComplete string) {
	fmt.Println("\n---- OAuth2 Device Code Flow ----")
	if verURIComplete != "" {
		fmt.Print(verURIComplete, "\n\nOR\n\n")
	}
	fmt.Println("Enter code at", verURI)
	fmt.Printf("\n\t%s\n\n", userCode)
}

type tmiSlog struct {
	l *slog.Logger
}

func (l *tmiSlog) Error(err error) { l.l.Error("TMI error", slog.String("err", err.Error())) }
func (l *tmiSlog) Status(s string) { l.l.Info("TMI status", slog.String("message", s)) }
func (l *tmiSlog) Send(s string)   { l.l.Debug("TMI send", slog.String("message", s)) }
func (l *tmiSlog) Recv(s string)   { l.l.Debug("TMI recv", slog.String("message", s)) }
func (l *tmiSlog) Ping(s string) {
	l.l.Log(context.Background(), slog.LevelDebug-1, "TMI ping", slog.String("message", s))
}
