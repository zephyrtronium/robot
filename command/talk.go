package command

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"regexp"
	"strconv"
	"time"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/message"
)

func speakCmd(ctx context.Context, robo *Robot, call *Invocation, effect string) string {
	if call.Message.Time().Before(call.Channel.SilentTime()) {
		robo.Log.InfoContext(ctx, "silent", slog.Time("until", call.Channel.SilentTime()))
		return ""
	}
	// Don't continue prompts that look like they start with TMI commands
	// (even though those don't do anything anymore).
	if ngPrompt.MatchString(call.Args["prompt"]) {
		robo.Log.WarnContext(ctx, "nasty prompt",
			slog.String("in", call.Channel.Name),
			slog.String("prompt", call.Args["prompt"]),
		)
		e := call.Channel.Emotes.Pick(rand.Uint32())
		return "no " + e
	}
	start := time.Now()
	m, trace, err := brain.Think(ctx, robo.Brain, call.Channel.Send, call.Args["prompt"])
	cost := time.Since(start)
	if err != nil {
		robo.Log.ErrorContext(ctx, "couldn't think", "err", err.Error())
		return ""
	}
	if m == "" {
		robo.Log.InfoContext(ctx, "thought nothing", slog.String("tag", call.Channel.Send), slog.String("prompt", call.Args["prompt"]))
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
	robo.Metrics.SpeakLatency.Observe(time.Since(start).Seconds(), call.Channel.Send, strconv.FormatBool(len(call.Args["prompt"]) == 0))
	robo.Metrics.UsedMessagesForGeneration.Observe(float64(len(trace)))
	robo.Log.InfoContext(ctx, "speak", "in", call.Channel.Name, "text", m, "emote", e)
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
	call.Channel.Message(ctx, message.Sent{Text: u})
}

// OwO genyewates an uwu message.
//   - prompt: Start of the message to use. Optional.
func OwO(ctx context.Context, robo *Robot, call *Invocation) {
	u := speakCmd(ctx, robo, call, "cmd OwO")
	if u == "" {
		return
	}
	u = lenlimit(owoize(u), 450)
	call.Channel.Message(ctx, message.Sent{Text: u})
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
	call.Channel.Message(ctx, message.Sent{Text: u})
}

// Hte generats an typod message.
//   - prompt: Start of the message to use. Optional.
func Hte(ctx context.Context, robo *Robot, call *Invocation) {
	u := speakCmd(ctx, robo, call, "cmd hte")
	if u == "" {
		return
	}
	u = lenlimit(hteize(u), 450)
	call.Channel.Message(ctx, message.Sent{Text: u})
}

// Rawr says rawr.
func Rawr(ctx context.Context, robo *Robot, call *Invocation) {
	if call.Message.Time().Before(call.Channel.SilentTime()) {
		robo.Log.InfoContext(ctx, "silent", slog.Time("until", call.Channel.SilentTime()))
		return
	}
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
	call.Channel.Message(ctx, message.Format("rawr %s", e).AsReply(call.Message.ID))
}

// HappyBirthdayToYou wishes the robot a happy birthday.
func HappyBirthdayToYou(ctx context.Context, robo *Robot, call *Invocation) {
	if call.Message.Time().Before(call.Channel.SilentTime()) {
		robo.Log.InfoContext(ctx, "silent", slog.Time("until", call.Channel.SilentTime()))
		return
	}
	e := call.Channel.Emotes.Pick(rand.Uint32())
	if e == "" {
		e = ":3"
	}
	t := time.Now()
	r := call.Channel.Rate.ReserveN(t, 1)
	if d := r.DelayFrom(t); d > 0 {
		robo.Log.InfoContext(ctx, "rate limited",
			slog.String("action", "birthday"),
			slog.String("in", call.Channel.Name),
			slog.String("delay", d.String()),
		)
		r.CancelAt(t)
		return
	}
	var m message.Sent
	switch t.Month() {
	case time.January:
		m = message.Format("No no no my birthday is next month. %s", e)
	case time.February:
		switch t.Day() {
		case 1, 2, 3, 4, 5:
			m = message.Format("Oh, but my birthday is later this month. %s", e)
		case 6:
			m = message.Format("My birthday is just a week away! I am so excited about this information. %s", e)
		case 7, 8, 9, 10:
			m = message.Format("My birthday is still less than a week away. %s", e)
		case 11:
			m = message.Format("Two days away...! %s", e)
		case 12:
			m = message.Format("My birthday is tomorrow! At least in my timezone. %s", e)
		case 13:
			m = message.Format("Thank you! Happy my birthday to you, too! %s", e)
		case 14:
			m = message.Format("You missed it. My birthday was yesterday. You are disqualified from being my valentine. %s", e)
		case 15, 16, 17, 18, 19, 20:
			m = message.Format("My birthday was the other day, actually, but I appreciate the sentiment. %s", e)
		default:
			m = message.Format("My birthday was earlier this month, actually, but I appreciate the sentiment. %s", e)
		}
	case time.March:
		m = message.Format("No no no my birthday was last month. %s", e)
	default:
		m = message.Format("My birthday is in February, silly %s", e)
	}
	call.Channel.Message(ctx, m.AsReply(call.Message.ID))
}

// Source gives a link to the source code.
func Source(ctx context.Context, robo *Robot, call *Invocation) {
	const srcMessage = `My source code is at https://github.com/zephyrtronium/robot – ` +
		`I'm written in Go, and I'm free, open-source software licensed ` +
		`under the GNU General Public License, Version 3.`
	call.Channel.Message(ctx, message.Sent{Reply: call.Message.ID, Text: srcMessage})
}

// Who describes Robot.
func Who(ctx context.Context, robo *Robot, call *Invocation) {
	const whoMessage = `I'm a Markov chain bot! I learn from things people say in chat, then spew vaguely intelligible memes back. More info at: https://github.com/zephyrtronium/robot#how-robot-works %s`
	e := call.Channel.Emotes.Pick(rand.Uint32())
	call.Channel.Message(ctx, message.Format(whoMessage, e).AsReply(call.Message.ID))
}

// Contact gives information on how to contact the bot owner.
func Contact(ctx context.Context, robo *Robot, call *Invocation) {
	e := call.Channel.Emotes.Pick(rand.Uint32())
	call.Channel.Message(ctx, message.Format("My operator is %[1]s. %[2]s is the best way to contact %[1]s. %[3]s", robo.Owner, robo.Contact, e).AsReply(call.Message.ID))
}
