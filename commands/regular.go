package commands

import (
	"context"
	"strings"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/irc"
)

func talk(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	if !br.ShouldTalk(ctx, msg, false) {
		return
	}
	with := strings.TrimSpace(matches[1])
	toks := brain.Tokens(with)
	m := br.TalkIn(ctx, msg.To(), toks)
	if m == "" {
		return
	}
	selsend(ctx, send, msg.Reply(m))
}

func talkCatchall(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	if !br.ShouldTalk(ctx, msg, false) {
		return
	}
	selsend(ctx, send, msg.Reply(br.TalkIn(ctx, msg.To(), nil)))
}

func uwu(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	if !br.ShouldTalk(ctx, msg, false) {
		return
	}
	m := br.TalkIn(ctx, msg.To(), nil)
	if m == "" {
		return
	}
	selsend(ctx, send, msg.Reply(uwuRep.Replace(m)))
}

var uwuRep = strings.NewReplacer(
	"r", "w", "R", "W",
	"l", "w", "L", "W",
	"na", "nya", "Na", "Nya", "NA", "NYA",
	"ni", "nyi", "Ni", "Nyi", "NI", "NYI",
	"nu", "nyu", "Nu", "Nyu", "NU", "NYU",
	"ne", "nye", "Ne", "Nye", "NE", "NYE",
	"no", "nyo", "No", "Nyo", "NO", "NYO",
)
