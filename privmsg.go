package main

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"gitlab.com/zephyrtronium/tmi"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/command"
	"github.com/zephyrtronium/robot/message"
	"github.com/zephyrtronium/robot/privacy"
	"github.com/zephyrtronium/robot/userhash"
)

// tmiMessage processes a PRIVMSG from TMI.
func (robo *Robot) tmiMessage(ctx context.Context, send chan<- *tmi.Message, msg *tmi.Message) {
	ch := robo.channels[msg.To()]
	if ch == nil {
		// TMI gives a WHISPER for a direct message, so this is a message to a
		// channel that isn't configured. Ignore it.
		return
	}
	// Run the rest in a worker so that we don't block the message loop.
	work := func(ctx context.Context) {
		m := message.FromTMI(msg)
		from := m.Sender
		if ch.Ignore[from] {
			slog.DebugContext(ctx, "message from ignored user", slog.String("in", ch.Name))
			return
		}
		if ch.Block.MatchString(m.Text) {
			slog.InfoContext(ctx, "blocked message", slog.String("in", ch.Name))
			return
		}
		if cmd, ok := parseCommand(robo.tmi.me, m.Text); ok {
			robo.command(ctx, ch, m, from, cmd)
			return
		}
		robo.learn(ctx, ch, userhash.New(robo.secrets.userhash), m)
		// TODO(zeph): this should be asking for a reservation
		if !ch.Rate.Allow() {
			return
		}
		switch err := ch.Memery.Check(m.Time(), from, m.Text); err {
		case channel.ErrNotCopypasta: // do nothing
		case nil:
			// Meme detected. Copypasta.
			text := m.Text
			// TODO(zeph): effects; once we apply them, we also need to check block
			msg := message.Format("", ch.Name, "%s", text)
			robo.sendTMI(ctx, send, msg)
			return
		default:
			slog.ErrorContext(ctx, "failed copypasta check", slog.String("err", err.Error()), slog.Any("message", m))
			// Continue on.
		}
		if rand.Float64() > ch.Responses {
			return
		}
		s, err := brain.Speak(ctx, robo.brain, ch.Send, "")
		if err != nil {
			slog.ErrorContext(ctx, "wanted to speak but failed", slog.String("err", err.Error()))
			return
		}
		e := ch.Emotes.Pick(rand.Uint32())
		s = strings.TrimSpace(s + " " + e)
		// TODO(zeph): effect
		if ch.Block.MatchString(s) {
			slog.WarnContext(ctx, "wanted to send blocked message", slog.String("in", ch.Name), slog.String("text", s))
			return
		}
		msg := message.Format("", ch.Name, "%s", s)
		robo.sendTMI(ctx, send, msg)
	}
	robo.enqueue(ctx, work)
}

func (robo *Robot) command(ctx context.Context, ch *channel.Channel, m *message.Received, from, cmd string) {
	if from == robo.tmi.owner {
		// TODO(zeph): check owner and moderator commands
	}
	if ch.Mod[from] || m.IsModerator {
		// TODO(zeph): check moderator commands
	}
	c, args := findTwitch(twitchAny, cmd)
	if c == nil {
		return
	}
	slog.InfoContext(ctx, "regular command", slog.String("name", c.name), slog.Any("args", args))
	r := command.Robot{
		Log:      slog.Default(),
		Channels: robo.channels,
		Brain:    robo.brain,
		Privacy:  robo.privacy,
	}
	inv := command.Invocation{
		Channel: ch,
		Message: m,
		Args:    args,
		Hasher:  userhash.New(robo.secrets.userhash),
	}
	c.fn(ctx, &r, &inv)
}

func (robo *Robot) enqueue(ctx context.Context, work func(context.Context)) {
	var w chan func(context.Context)
	// Get a worker if one exists. Otherwise, spawn a new one.
	select {
	case w = <-robo.works:
	default:
		w = make(chan func(context.Context), 1)
		go worker(ctx, robo.works, w)
	}
	// Send it work.
	select {
	case <-ctx.Done():
		return
	case w <- work:
	}
}

// worker runs works for a while. The provided context is passed to each work.
func worker(ctx context.Context, works chan chan func(context.Context), ch chan func(context.Context)) {
	for {
		select {
		case <-ctx.Done():
			return
		case work := <-ch:
			work(ctx)
			// Replace ourselves in the pool if it needs additional capacity.
			// Otherwise, we're done.
			select {
			case works <- ch:
			default:
				return
			}
		}
	}
}

// learn learns a given message's text if it passes ch's filters.
func (robo *Robot) learn(ctx context.Context, ch *channel.Channel, hasher userhash.Hasher, msg *message.Received) {
	if !ch.Enabled {
		return
	}
	switch err := robo.privacy.Check(ctx, msg.Sender); err {
	case nil: // do nothing
	case privacy.ErrPrivate:
		slog.DebugContext(ctx, "private sender", slog.String("in", ch.Name))
		return
	default:
		slog.ErrorContext(ctx, "failed to check privacy", slog.String("err", err.Error()), slog.String("in", ch.Name))
		return
	}
	if ch.Block.MatchString(msg.Text) {
		slog.DebugContext(ctx, "blocked message", slog.String("in", ch.Name), slog.String("text", msg.Text))
		return
	}
	if ch.Learn == "" {
		slog.DebugContext(ctx, "no learn tag", slog.String("in", ch.Name))
		return
	}
	id, err := uuid.Parse(msg.ID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to parse message id", slog.String("err", err.Error()), slog.String("id", msg.ID))
		// Continue on with a zero UUID.
	}
	user := hasher.Hash(new(userhash.Hash), msg.Sender, msg.To, msg.Time())
	meta := &brain.MessageMeta{
		ID:   id,
		User: *user,
		Tag:  ch.Learn,
		Time: msg.Time(),
	}
	if err := brain.Learn(ctx, robo.brain, meta, brain.Tokens(nil, msg.Text)); err != nil {
		slog.ErrorContext(ctx, "failed to learn", slog.String("err", err.Error()))
	}
}

// sendTMI sends a message to TMI after waiting for the global rate limit.
// The caller should verify that it is safe to send the message.
func (robo *Robot) sendTMI(ctx context.Context, send chan<- *tmi.Message, msg message.Sent) {
	if err := robo.tmi.rate.Wait(ctx); err != nil {
		return
	}
	resp := message.ToTMI(msg)
	select {
	case <-ctx.Done():
		return
	case send <- resp:
	}
}

func parseCommand(name, text string) (string, bool) {
	text = strings.TrimSpace(text)
	text, _ = strings.CutPrefix(text, "@")
	// TODO(zeph): not quite right if our name contains one of those handful of
	// code points that has a different size between cases
	if len(text) < len(name) {
		return "", false
	}
	if strings.EqualFold(text[:len(name)], name) {
		text = text[len(name):]
		r, _ := utf8.DecodeRuneInString(text)
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			// Our name is a prefix of a word.
			return "", false
		}
		// This is a command. Skip to the next whitespace to get the text. If
		// there is no whitespace, the text is empty.
		k := strings.IndexFunc(text, unicode.IsSpace)
		if k < 0 {
			k = len(text)
		}
		return strings.TrimSpace(text[k:]), true
	}
	if strings.EqualFold(text[len(text)-len(name):], name) {
		text = text[:len(text)-len(name)]
		r, _ := utf8.DecodeLastRuneInString(text)
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			// Our name is a suffix of a word.
			return "", false
		}
		// This is a command. Trim off after the preceding whitespace to get
		// the text. Even though we already checked the start-of-text case,
		// there can still be no preceding whitespace in a case like "...name".
		k := strings.LastIndexFunc(text, unicode.IsSpace)
		if k < 0 {
			k = 0
		}
		return strings.TrimSpace(text[:k]), true
	}
	return "", false
}

type twitchCommand struct {
	parse *regexp.Regexp
	fn    command.Func
	name  string
}

func findTwitch(cmds []twitchCommand, text string) (*twitchCommand, map[string]string) {
	for i := range cmds {
		c := &cmds[i]
		u := c.parse.FindStringSubmatch(text)
		switch len(u) {
		case 0:
			continue
		case 1:
			return c, nil
		default:
			m := make(map[string]string, len(u)-1)
			s := c.parse.SubexpNames()
			for k, v := range u[1:] {
				m[s[k+1]] = v
			}
			return c, m
		}
	}
	return nil, nil
}

var twitchAny = []twitchCommand{
	{
		parse: regexp.MustCompile(`^(?i:say|generate)?\s*(?i:something)?\s*(?i:starting)?\s*(?i:with)?\s*(?<prompt>.*)`),
		fn:    command.Speak,
		name:  "speak",
	},
}
