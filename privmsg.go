package main

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gitlab.com/zephyrtronium/tmi"
	"golang.org/x/sync/errgroup"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/command"
	"github.com/zephyrtronium/robot/message"
	"github.com/zephyrtronium/robot/privacy"
	"github.com/zephyrtronium/robot/userhash"
)

// tmiMessage processes a PRIVMSG from TMI.
func (robo *Robot) tmiMessage(ctx context.Context, group *errgroup.Group, send chan<- *tmi.Message, msg *tmi.Message) {
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
		if cmd, ok := parseCommand(robo.tmi.name, m.Text); ok {
			robo.command(ctx, ch, m, from, cmd)
			return
		}
		ch.History.Add(m.ID, m.Sender, m.Text)
		// If the message is a reply to e.g. Bocchi, TMI adds @Bocchi to the
		// start of the message text.
		// That's helpful for commands, which we've already processed, but
		// otherwise we probably don't want to see it. Remove it.
		if _, ok := msg.Tag("reply-parent-msg-id"); ok && strings.HasPrefix(m.Text, "@") {
			at, t, _ := strings.Cut(m.Text, " ")
			slog.DebugContext(ctx, "stripped reply mention", slog.String("mention", at), slog.String("text", t))
			m.Text = t
		}
		robo.learn(ctx, ch, userhash.New(robo.secrets.userhash), m)
		switch err := ch.Memery.Check(m.Time(), from, m.Text); err {
		case channel.ErrNotCopypasta: // do nothing
		case nil:
			// Meme detected. Copypasta.
			t := time.Now()
			r := ch.Rate.ReserveN(t, 1)
			if d := r.DelayFrom(t); d > 0 {
				// But we can't meme it. Restore it so we can next time.
				slog.InfoContext(ctx, "won't copypasta; rate limited",
					slog.String("action", "copypasta"),
					slog.String("in", ch.Name),
					slog.String("delay", d.String()),
				)
				ch.Memery.Unblock(m.Text)
				r.CancelAt(t)
				return
			}
			text := m.Text
			f := ch.Effects.Pick(rand.Uint32())
			s := command.Effect(f, text)
			ch.Memery.Block(m.Time(), s)
			if ch.Block.MatchString(s) {
				// Don't send things we wouldn't learn.
				slog.InfoContext(ctx, "won't copypasta blocked message", slog.String("message", s), slog.String("effect", f))
				r.CancelAt(t)
				break
			}
			slog.InfoContext(ctx, "copypasta", slog.String("message", s), slog.String("effect", f))
			msg := message.Format("", ch.Name, "%s", s)
			robo.sendTMI(ctx, send, msg)
			return
		default:
			slog.ErrorContext(ctx, "failed copypasta check", slog.String("err", err.Error()), slog.Any("message", m))
			// Continue on.
		}
		if rand.Float64() > ch.Responses {
			return
		}
		start := time.Now()
		s, trace, err := brain.Speak(ctx, robo.brain, ch.Send, "")
		cost := time.Since(start)
		if err != nil {
			slog.ErrorContext(ctx, "wanted to speak but failed", slog.String("err", err.Error()))
			return
		}
		if s == "" {
			slog.InfoContext(ctx, "spoke nothing", slog.String("tag", ch.Send))
			return
		}
		x := rand.Uint64()
		e := ch.Emotes.Pick(uint32(x))
		f := ch.Effects.Pick(uint32(x >> 32))
		slog.InfoContext(ctx, "speak", slog.String("text", s), slog.String("emote", e), slog.String("effect", f))
		se := strings.TrimSpace(s + " " + e)
		sef := command.Effect(f, se)
		if err := robo.spoken.Record(ctx, ch.Send, sef, trace, time.Now(), cost, s, e, f); err != nil {
			slog.ErrorContext(ctx, "record trace failed", slog.Any("err", err))
			return
		}
		if ch.Block.MatchString(se) || ch.Block.MatchString(sef) {
			slog.WarnContext(ctx, "wanted to send blocked message", slog.String("in", ch.Name), slog.String("text", sef))
			return
		}
		// Now that we've done all the work, which might take substantial time,
		// check whether we can use it.
		t := time.Now()
		r := ch.Rate.ReserveN(t, 1)
		if d := r.DelayFrom(t); d > 0 {
			slog.InfoContext(ctx, "won't speak; rate limited",
				slog.String("action", "copypasta"),
				slog.String("in", ch.Name),
				slog.String("delay", d.String()),
			)
			r.CancelAt(t)
			return
		}
		msg := message.Format("", ch.Name, "%s", sef)
		robo.sendTMI(ctx, send, msg)
	}
	robo.enqueue(ctx, group, work)
}

func (robo *Robot) command(ctx context.Context, ch *channel.Channel, m *message.Received, from, cmd string) {
	var c *twitchCommand
	var args map[string]string
	level := "any"
	switch {
	case from == robo.tmi.owner:
		c, args = findTwitch(twitchOwner, cmd)
		if c != nil {
			level = "owner"
			break
		}
		fallthrough
	case ch.Mod[from], m.IsModerator:
		c, args = findTwitch(twitchMod, cmd)
		if c != nil {
			level = "mod"
			break
		}
		fallthrough
	default:
		c, args = findTwitch(twitchAny, cmd)
	}
	if c == nil {
		return
	}
	slog.InfoContext(ctx, "command", slog.String("level", level), slog.String("name", c.name), slog.Any("args", args))
	r := command.Robot{
		Log:      slog.Default(),
		Channels: robo.channels,
		Brain:    robo.brain,
		Privacy:  robo.privacy,
		Spoken:   robo.spoken,
	}
	inv := command.Invocation{
		Channel: ch,
		Message: m,
		Args:    args,
		Hasher:  userhash.New(robo.secrets.userhash),
	}
	c.fn(ctx, &r, &inv)
}

func (robo *Robot) enqueue(ctx context.Context, group *errgroup.Group, work func(context.Context)) {
	var w chan func(context.Context)
	// Get a worker if one exists. Otherwise, spawn a new one.
	select {
	case w = <-robo.works:
	default:
		w = make(chan func(context.Context), 1)
		group.Go(func() error {
			worker(ctx, robo.works, w)
			return nil
		})
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
	if !ch.Enabled.Load() {
		slog.DebugContext(ctx, "not learning in disabled channel", slog.String("in", ch.Name))
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
	user := hasher.Hash(new(userhash.Hash), msg.Sender, msg.To, msg.Time())
	if err := brain.Learn(ctx, robo.brain, ch.Learn, msg.ID, *user, msg.Time(), brain.Tokens(nil, msg.Text)); err != nil {
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

var twitchOwner = []twitchCommand{
	{
		parse: regexp.MustCompile(`^(?i:in\s+(?<in>#\S+)[,:]?\s+echo)\s+(?<msg>.*)`),
		fn:    command.EchoIn,
		name:  "echo-in",
	},
}

var twitchMod = []twitchCommand{
	{
		parse: regexp.MustCompile(`^(?i:echo)\s+(?<msg>.*)`),
		fn:    command.Echo,
		name:  "echo",
	},
	{
		parse: regexp.MustCompile(`(?i)^(?:tell\s+me|talk)?\s*(?:about)?\s*(?:ranked)?\s*(?:competitive)?\s*marriage`),
		fn:    command.DescribeMarriage,
		name:  "describe-marriage",
	},
}

var twitchAny = []twitchCommand{
	{
		parse: regexp.MustCompile(`^(?i:give\s+me\s+privacy|ignore\s+me)`),
		fn:    command.Private,
		name:  "private",
	},
	{
		parse: regexp.MustCompile(`(?i)^(?:you\s+(?:can|may)\s+)?learn\s+from\s+me(?:\s+again)?|invade\s+my\s+privacy`),
		fn:    command.Unprivate,
		name:  "unprivate",
	},
	{
		parse: regexp.MustCompile(`(?i)^what\s+(?:info(?:rmation)?\s+)do\s+you\s+(?:collect|store)`),
		fn:    command.DescribePrivacy,
		name:  "describe-privacy",
	},
	{
		parse: regexp.MustCompile(`(?i)^[¿¡]*\s*(?:ple?a?se?\s+)?(?:will\s+y?o?u\s+)?(?:\s*ple?a?se?\s+)?(?:marry\s+me|be?\s+my\s+(?<partnership>wife|waifu|h[ua]su?bando?|partner|spouse|daddy|mommy))`),
		fn:    command.Marry,
		name:  "marry",
	},
	{
		parse: regexp.MustCompile(`^how\s+much\s+do\s+you\s+(?:like|love|luv)\s+me`),
		fn:    command.Affection,
		name:  "affection",
	},
	{
		parse: regexp.MustCompile(`^(?i:OwO|uwu)`),
		fn:    command.OwO,
		name:  "OwO",
	},
	{
		parse: regexp.MustCompile(`^(?i:how\s*[a']?re?\s+y?o?u?)|^A(?:A|\s)+$`),
		fn:    command.AAAAA,
		name:  "AAAAA",
	},
	{
		parse: regexp.MustCompile(`^(?i:r+o+a+r+|r+a+w+r+)`),
		fn:    command.Rawr,
		name:  "rawr",
	},
	{
		parse: regexp.MustCompile(`^(?i:where(?:'?s|\s+is)?\s+y?o?u'?re?\s+so?u?rce?(?:\s*code)?)`),
		fn:    command.Source,
		name:  "source",
	},
	{
		parse: regexp.MustCompile(`(?i)^[¿¡]*\s*(?:who\s+a?re?\s+y?o?u|how\s+do\s+y?o?u\s+w[oe]?rk)`),
		fn:    command.Who,
		name:  "who",
	},
	{
		// NOTE(zeph): This command MUST be last, because it swallows all invocations.
		parse: regexp.MustCompile(`^(?i:say|generate)\s*(?i:something)?\s*(?i:starting)?\s*(?i:with)?\s*(?<prompt>.*)|`),
		fn:    command.Speak,
		name:  "speak",
	},
}
