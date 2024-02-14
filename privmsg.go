package main

import (
	"context"
	"log"
	"math/rand/v2"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"gitlab.com/zephyrtronium/tmi"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/channel"
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
			return
		}
		if cmd, ok := parseCommand(robo.tmi.me, m.Text); ok {
			if from == robo.tmi.owner {
				// TODO(zeph): check owner and moderator commands
			}
			if ch.Mod[from] || m.IsModerator {
				// TODO(zeph): check moderator commands
			}
			// TODO(zeph): check regular commands
			_ = cmd
			return
		}
		robo.learn(ctx, ch, userhash.New(robo.secrets.userhash), m)
		if !ch.Rate.Allow() {
			return
		}
		if err := ch.Memery.Check(m.Time(), from, m.Text); err == nil {
			// NOTE(zeph): inverted error check
			robo.sendTMI(ctx, send, ch, m.Text)
			return
		} else if err != channel.ErrNotCopypasta {
			log.Println("copypasta error:", err)
		}
		if rand.Float64() < ch.Responses {
			s, err := brain.Speak(ctx, robo.brain, ch.Send, "")
			if err != nil {
				log.Println("speak error:", err)
				return
			}
			e := ch.Emotes.Pick(rand.Uint32())
			s = strings.TrimSpace(s + " " + e)
			// TODO(zeph): effect
			robo.sendTMI(ctx, send, ch, s)
		}
	}
	robo.enqueue(ctx, work)
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
func (robo *Robot) learn(ctx context.Context, ch *channel.Channel, hasher userhash.Hasher, msg *message.Message) {
	if err := robo.privacy.Check(ctx, msg.Sender); err != nil {
		if err == privacy.ErrPrivate {
			// TODO(zeph): log at a lower priority level
		}
		// TODO(zeph): log
		return
	}
	if ch.Block.MatchString(msg.Text) {
		return
	}
	if ch.Learn == "" {
		return
	}
	// Ignore the error. If we get a bad one, we'll record a zero UUID.
	// TODO(zeph): log error instead
	id, _ := uuid.Parse(msg.ID)
	user := hasher.Hash(new(userhash.Hash), msg.Sender, msg.To, msg.Time())
	meta := &brain.MessageMeta{
		ID:   id,
		User: *user,
		Tag:  ch.Learn,
		Time: msg.Time(),
	}
	if err := brain.Learn(ctx, robo.brain, meta, brain.Tokens(nil, msg.Text)); err != nil {
		// TODO(zeph): log
	}
}

// sendTMI sends a message to TMI after waiting for the global rate limit.
func (robo *Robot) sendTMI(ctx context.Context, send chan<- *tmi.Message, ch *channel.Channel, s string) {
	if ch.Block.MatchString(s) {
		return
	}
	if err := robo.tmi.rate.Wait(ctx); err != nil {
		return
	}
	resp := &tmi.Message{
		Command:  "PRIVMSG",
		Params:   []string{ch.Name},
		Trailing: s,
	}
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
