package command

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"regexp"
	"time"

	"github.com/zephyrtronium/robot/brain"
)

func speakCmd(ctx context.Context, robo *Robot, call *Invocation, effect string) string {
	// Don't continue prompts that look like they start with TMI commands
	// (even though those don't do anything anymore).
	if ngPrompt.MatchString(call.Args["prompt"]) {
		robo.Log.WarnContext(ctx, "nasty prompt",
			slog.String("in", call.Channel.Name),
			slog.String("from", call.Message.Name),
			slog.String("prompt", call.Args["prompt"]),
		)
		e := call.Channel.Emotes.Pick(rand.Uint32())
		return "no " + e
	}
	start := time.Now()
	m, trace, err := brain.Speak(ctx, robo.Brain, call.Channel.Send, call.Args["prompt"])
	cost := time.Since(start)
	if err != nil {
		robo.Log.ErrorContext(ctx, "couldn't speak", "err", err.Error())
		return ""
	}
	if m == "" {
		slog.InfoContext(ctx, "spoke nothing", slog.String("tag", call.Channel.Send), slog.String("prompt", call.Args["prompt"]))
		return ""
	}
	e := call.Channel.Emotes.Pick(rand.Uint32())
	s := m + " " + e
	if err := robo.Spoken.Record(ctx, call.Channel.Send, s, trace, call.Message.Time(), cost, m, e, effect); err != nil {
		robo.Log.ErrorContext(ctx, "couldn't record trace", slog.Any("err", err))
		return ""
	}
	if call.Channel.Block.MatchString(s) {
		robo.Log.WarnContext(ctx, "generated blocked message",
			slog.String("in", call.Channel.Name),
			slog.String("text", m),
			slog.String("emote", e),
		)
		return ""
	}
	t := time.Now()
	r := call.Channel.Rate.ReserveN(t, 1)
	if d := r.DelayFrom(t); d > 0 {
		robo.Log.InfoContext(ctx, "won't speak; rate limited",
			slog.String("action", "command"),
			slog.String("in", call.Channel.Name),
			slog.String("delay", d.String()),
		)
		r.CancelAt(t)
		return ""
	}
	// block the generated message from being later recognized as a meme.
	call.Channel.Memery.Block(call.Message.Time(), s)
	slog.InfoContext(ctx, "speak", "in", call.Channel.Name, "text", m, "emote", e)
	return m + " " + e
}

var ngPrompt = regexp.MustCompile(`^/|^\.\w`)

// Speak generates a message.
//   - prompt: Start of the message to use. Optional.
func Speak(ctx context.Context, robo *Robot, call *Invocation) {
	u := speakCmd(ctx, robo, call, "")
	if u == "" {
		return
	}
	u = lenlimit(u, 450)
	call.Channel.Message(ctx, "", u)
}

// OwO genyewates an uwu message.
//   - prompt: Start of the message to use. Optional.
func OwO(ctx context.Context, robo *Robot, call *Invocation) {
	u := speakCmd(ctx, robo, call, "cmd OwO")
	if u == "" {
		return
	}
	u = lenlimit(owoize(u), 450)
	call.Channel.Message(ctx, "", u)
}

// AAAAA AAAAAAAAA A AAAAAAA.
func AAAAA(ctx context.Context, robo *Robot, call *Invocation) {
	// Never use a prompt for this one.
	// But also look before we delete in case the arg isn't given.
	if call.Args["prompt"] != "" {
		delete(call.Args, "prompt")
	}
	u := speakCmd(ctx, robo, call, "cmd AAAAA")
	if u == "" {
		return
	}
	u = lenlimit(aaaaaize(u), 40)
	call.Channel.Message(ctx, "", u)
}

// Rawr says rawr.
func Rawr(ctx context.Context, robo *Robot, call *Invocation) {
	e := call.Channel.Emotes.Pick(rand.Uint32())
	if e == "" {
		e = ":3"
	}
	t := time.Now()
	r := call.Channel.Rate.ReserveN(t, 1)
	if d := r.DelayFrom(t); d > 0 {
		robo.Log.InfoContext(ctx, "won't rawr; rate limited",
			slog.String("action", "rawr"),
			slog.String("in", call.Channel.Name),
			slog.String("delay", d.String()),
		)
		r.CancelAt(t)
		return
	}
	call.Channel.Message(ctx, call.Message.ID, "rawr "+e)
}

// Source gives a link to the source code.
func Source(ctx context.Context, robo *Robot, call *Invocation) {
	const srcMessage = `My source code is at https://github.com/zephyrtronium/robot – ` +
		`I'm written in Go, and I'm free, open-source software licensed ` +
		`under the GNU General Public License, Version 3.`
	call.Channel.Message(ctx, call.Message.ID, srcMessage)
}

// Who describes Robot.
func Who(ctx context.Context, robo *Robot, call *Invocation) {
	const whoMessage = `I'm a Markov chain bot! I learn from things people say in chat, then spew vaguely intelligible memes back. More info at: https://github.com/zephyrtronium/robot#how-robot-works`
	e := call.Channel.Emotes.Pick(rand.Uint32())
	call.Channel.Message(ctx, call.Message.ID, whoMessage+" "+e)
}
