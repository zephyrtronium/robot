package command

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"strings"
	"unicode"

	"github.com/zephyrtronium/robot/brain"
)

func speakCmd(ctx context.Context, robo *Robot, call *Invocation) string {
	t := call.Message.Time()
	r := call.Channel.Rate.ReserveN(call.Message.Time(), 1)
	cancel := func() { r.CancelAt(t) }
	if d := r.DelayFrom(t); d > 0 {
		robo.Log.InfoContext(ctx, "won't speak; rate limited", slog.String("delay", d.String()))
		cancel()
		return ""
	}
	// TODO(zeph): record trace
	m, _, err := brain.Speak(ctx, robo.Brain, call.Channel.Send, call.Args["prompt"])
	if err != nil {
		robo.Log.ErrorContext(ctx, "couldn't speak", "err", err.Error())
		cancel()
		return ""
	}
	if call.Channel.Block.MatchString(m) {
		robo.Log.WarnContext(ctx, "generated blocked message",
			slog.String("in", call.Channel.Name),
			slog.String("text", m),
		)
		cancel()
		return ""
	}
	e := call.Channel.Emotes.Pick(rand.Uint32())
	slog.InfoContext(ctx, "speak", "in", call.Channel.Name, "text", m, "emote", e)
	return m + " " + e
}

func Speak(ctx context.Context, robo *Robot, call *Invocation) {
	u := speakCmd(ctx, robo, call)
	if u == "" {
		return
	}
	if len(u) > 450 {
		r := []rune(u)
		r = r[:min(450, len(r))]
		u = string(r)
	}
	call.Channel.Message(ctx, "", u)
}

func OwO(ctx context.Context, robo *Robot, call *Invocation) {
	u := speakCmd(ctx, robo, call)
	if u == "" {
		return
	}
	u = owoRep.Replace(u)
	if len(u) > 450 {
		r := []rune(u)
		r = r[:min(450, len(r))]
		u = string(r)
	}
	call.Channel.Message(ctx, "", u)
}

var owoRep = strings.NewReplacer(
	"r", "w", "R", "W",
	"l", "w", "L", "W",
	"na", "nya", "Na", "Nya", "NA", "NYA",
	"ni", "nyi", "Ni", "Nyi", "NI", "NYI",
	"nu", "nyu", "Nu", "Nyu", "NU", "NYU",
	"ne", "nye", "Ne", "Nye", "NE", "NYE",
	"no", "nyo", "No", "Nyo", "NO", "NYO",
)

func AAAAA(ctx context.Context, robo *Robot, call *Invocation) {
	u := speakCmd(ctx, robo, call)
	if u == "" {
		return
	}
	u = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return 'A'
		}
		return r
	}, u)
	if len(u) > 40 {
		r := []rune(u)
		r = r[:min(40, len(r))]
		u = string(r)
	}
	call.Channel.Message(ctx, "", u)
}

var (
	_ Func = Speak
	_ Func = OwO
	_ Func = AAAAA
)
