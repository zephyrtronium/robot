package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gitlab.com/zephyrtronium/tmi"
	"golang.org/x/oauth2"
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
	tmi *client[*tmi.Message, *tmi.Message]
}

// client is the settings for OAuth2 and related elements.
// The type parameter is the type of messages sent TO the service.
type client[Send, Receive any] struct {
	// send is the channel on which messages are sent.
	send chan Send
	// recv is the channel on which received messages are communicated.
	recv chan Receive
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
	tok, err := twitchToken(ctx, robo.tmi.tokens)
	if err != nil {
		return err
	}
	// TODO(zeph): resolve usernames in configs to user ids
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
	tmi.Connect(ctx, cfg, &tmiSlog{slog.Default()}, robo.tmi.send, robo.tmi.recv)
	return ctx.Err()
}

// twitchToken gets a valid Twitch access token by refreshing until it
// validates successfully.
func twitchToken(ctx context.Context, tokens auth.TokenSource) (*oauth2.Token, error) {
	tok, err := tokens.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't obtain access token for TMI login: %w", err)
	}
	for range 5 {
		err := validateTwitch(ctx, tok)
		switch {
		case err == nil:
			// Current token is good.
			return tok, nil
		case errors.Is(err, errNeedRefresh):
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

// validateTwitch validates a Twitch access token. If the returned error Is
// errNeedRefresh, then the caller should refresh it and try again.
func validateTwitch(ctx context.Context, tok *oauth2.Token) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		return fmt.Errorf("couldn't make validate request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't validate access token: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("couldn't read token validation response: %w", err)
	}
	var s struct {
		ClientID  string   `json:"client_id"`
		Login     string   `json:"login"`
		Scopes    []string `json:"scopes"`
		UserID    string   `json:"user_id"`
		ExpiresIn int      `json:"expires_in"`
		Message   string   `json:"message"`
		Status    int      `json:"status"`
	}
	if err := json.Unmarshal(body, &s); err != nil {
		return fmt.Errorf("couldn't unmarshal token validation response: %w", err)
	}
	slog.InfoContext(ctx, "token validation", slog.String("token", tok.AccessToken), slog.Any("result", s))
	if resp.StatusCode == http.StatusUnauthorized {
		// Token expired or otherwise invalid. We need a refresh.
		return fmt.Errorf("token validation failed: %s (%w)", s.Message, errNeedRefresh)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token validation failed: %s (%s)", s.Message, resp.Status)
	}
	return nil
}

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
				// TODO(zeph): forget messages from the target (even self)
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

var errNeedRefresh = errors.New("need refresh")
