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
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/zephyrtronium/robot/auth"
	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/metrics"
	"github.com/zephyrtronium/robot/privacy"
	"github.com/zephyrtronium/robot/spoken"
	"github.com/zephyrtronium/robot/syncmap"
	"github.com/zephyrtronium/robot/twitch"
	"github.com/zephyrtronium/robot/userhash"
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
	channels *syncmap.Map[string, *channel.Channel]
	// works is the worker queue.
	works chan chan func(context.Context)
	// hashes is a function that obtains userhashers.
	hashes func() userhash.Hasher
	// owner is the username of the owner.
	owner string
	// ownerContact describes contact information for the owner.
	ownerContact string
	// tmi contains the bot's Twitch OAuth2 settings. It may be nil if there is
	// no Twitch configuration.
	tmi *client[*tmi.Message, *tmi.Message]
	// twitch is the Twitch API client.
	twitch twitch.Client
	// Metrics are a collection of custom domain specific Metrics.
	Metrics *metrics.Metrics
}

// client is the settings for OAuth2 and related elements.
type client[Send, Receive any] struct {
	// send is the channel on which messages are sent.
	send chan Send
	// recv is the channel on which received messages are communicated.
	recv chan Receive
	// clientID is the OAuth2 application client ID.
	clientID string
	// name is the bot's username. The interpretation of this is domain-specific.
	name string
	// userID is the bot's user ID. The interpretation of this is domain-specific.
	userID string
	// owner is the user ID of the owner. The interpretation of this is
	// domain-specific.
	owner string
	// rate is the global rate limiter for messages sent to this client.
	rate *rate.Limiter
	// tokens is the source of OAuth2 tokens.
	tokens auth.TokenSource
}

// New creates a new robot instance.
func New(usersKey []byte, poolSize int) *Robot {
	return &Robot{
		channels: syncmap.New[string, *channel.Channel](),
		works:    make(chan chan func(context.Context), poolSize),
		hashes:   func() userhash.Hasher { return userhash.New(usersKey) },
		Metrics:  newMetrics(),
	}
}

func (robo *Robot) Run(ctx context.Context, listen string) error {
	group, ctx := errgroup.WithContext(ctx)
	// TODO(zeph): stdin?
	if robo.tmi != nil {
		group.Go(func() error { return robo.runTwitch(ctx, group) })
	}
	if listen != "" {
		group.Go(func() error { return robo.api(ctx, listen, new(http.ServeMux), robo.Metrics.Collectors()) })
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
	group.Go(func() error {
		robo.tmiLoop(ctx, group, robo.tmi.send, robo.tmi.recv)
		return nil
	})
	group.Go(func() error {
		return robo.twitchValidateLoop(ctx)
	})
	group.Go(func() error {
		return robo.streamsLoop(ctx, robo.channels)
	})
	tok, err := robo.tmi.tokens.Token(ctx)
	if err != nil {
		return err
	}
	for {
		cfg := tmi.ConnectConfig{
			Dial:         new(tls.Dialer).DialContext,
			RetryWait:    tmi.RetryList(true, 0, time.Second, time.Minute, 5*time.Minute),
			Nick:         strings.ToLower(robo.tmi.name),
			Pass:         "oauth:" + tok.AccessToken,
			Capabilities: []string{"twitch.tv/commands", "twitch.tv/tags"},
			Timeout:      300 * time.Second,
		}
		err = tmi.Connect(ctx, cfg, &tmiSlog{slog.Default()}, robo.tmi.send, robo.tmi.recv)
		switch {
		case err == nil:
			// We received a RECONNECT and exited normally. Do nothing.
			// It's likely (though not guaranteed) we'll need a refresh,
			// but we can worry about that when we're told to do it.
		case errors.Is(err, tmi.ErrAuthenticationFailed):
			tok, err = robo.tmi.tokens.Refresh(ctx, tok)
			if err != nil {
				return err
			}
		default:
			return err
		}
	}
}

func (robo *Robot) twitchValidateLoop(ctx context.Context) error {
	tm := time.NewTicker(time.Hour)
	defer tm.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tm.C: // continue below
		}
		tok, err := robo.tmi.tokens.Token(ctx)
		if err != nil {
			return fmt.Errorf("validation loop failed to get user access token: %w", err)
		}
		val, err := twitch.Validate(ctx, robo.twitch.HTTP, tok)
		switch {
		case err == nil:
			slog.InfoContext(ctx, "validation loop",
				slog.String("clientid", val.ClientID),
				slog.String("userid", val.UserID),
				slog.String("login", val.Login),
				slog.Int("expires", val.ExpiresIn),
			)
		case errors.Is(err, twitch.ErrNeedRefresh):
			_, err := robo.tmi.tokens.Refresh(ctx, tok)
			if err != nil {
				return fmt.Errorf("validation loop failed to refresh user access token: %w", err)
			}
		default:
			if val != nil {
				slog.ErrorContext(ctx, "validation loop", slog.Int("status", val.Status), slog.String("message", val.Message))
			}
			return fmt.Errorf("validation loop failed to validate user access token: %w", err)
		}
	}
}

func (robo *Robot) streamsLoop(ctx context.Context, channels *syncmap.Map[string, *channel.Channel]) error {
	// TODO(zeph): one day we should switch to eventsub
	// TODO(zeph): remove anything learned since the last check when offline
	tok, err := robo.tmi.tokens.Token(ctx)
	if err != nil {
		return err
	}
	streams := make([]twitch.Stream, 0, channels.Len())
	m := make(map[string]bool, channels.Len())
	// Run once at the start so we start learning in online streams immediately.
	streams = streams[:0]
	for _, ch := range channels.All() {
		n := strings.ToLower(strings.TrimPrefix(ch.Name, "#"))
		streams = append(streams, twitch.Stream{UserLogin: n})
	}
	for range 5 {
		// TODO(zeph): limit to 100
		streams, err = twitch.UserStreams(ctx, robo.twitch, tok, streams)
		switch {
		case err == nil:
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
			for _, ch := range channels.All() {
				n := strings.ToLower(strings.TrimPrefix(ch.Name, "#"))
				ch.Enabled.Store(m[n])
			}
		case errors.Is(err, twitch.ErrNeedRefresh):
			tok, err = robo.tmi.tokens.Refresh(ctx, tok)
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
			for _, ch := range channels.All() {
				n := strings.TrimPrefix(ch.Name, "#")
				streams = append(streams, twitch.Stream{UserLogin: n})
			}
			for range 5 {
				// TODO(zeph): limit to 100
				streams, err = twitch.UserStreams(ctx, robo.twitch, tok, streams)
				switch {
				case err == nil:
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
					for _, ch := range channels.All() {
						n := strings.ToLower(strings.TrimPrefix(ch.Name, "#"))
						ch.Enabled.Store(m[n])
					}
				case errors.Is(err, twitch.ErrNeedRefresh):
					tok, err = robo.tmi.tokens.Refresh(ctx, tok)
					if err != nil {
						slog.ErrorContext(ctx, "failed to refresh token", slog.Any("err", err))
						return fmt.Errorf("couldn't get valid access token: %w", err)
					}
					continue
				default:
					slog.ErrorContext(ctx, "failed to query online broadcasters", slog.Any("streams", streams), slog.Any("err", err))
					// Set all streams as offline.
					for _, ch := range channels.All() {
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
