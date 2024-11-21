package command

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"gitlab.com/zephyrtronium/pick"

	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/message"
)

func score(log *slog.Logger, h *channel.History[*message.Received[string]], user string) (x float64, c, f, l, n int) {
	mine := make(map[string]map[string]struct{})
	for m := range h.All() {
		who, text := m.Sender, m.Text
		n++
		// Count the number of distinct other users who said each of your messages
		// after you said it.
		m, ok := mine[text]
		if who != user {
			if ok {
				log.Debug("scoring meme", slog.String("user", user), slog.String("memer", who), slog.String("msg", text))
				m[who] = struct{}{}
				c += len(m) // n.b. quadratic growth for this component
			}
			continue
		}
		// Count the messages you sent.
		if !ok {
			log.Debug("scoring first", slog.String("user", user), slog.String("msg", text))
			m = map[string]struct{}{user: {}}
			mine[text] = m
			f++
		}
		// Get the length of the longest message you sent.
		l = max(l, len(text))
	}
	// Calculate score from components.
	// Participating in copypasta generally should matter more than generic
	// talking (even in the chats that don't copypasta, where this part just
	// tends toward zero).
	// Sending longer messages should matter less than frequency or memery.
	// There are other criteria we could add, like the number of distinct runes
	// used in messages you've sent or the uniqueness of your messages based on
	// a metric like tf-idf. However, this can't be too expensive to compute.
	x = float64(c*c)/float64(f+1) + float64(c*f+f) + math.Sqrt(float64(l))
	log.Info("user score",
		slog.String("user", user),
		slog.Int("msgs", n),
		slog.Int("freq", f),
		slog.Int("meme", c),
		slog.Int("len", l),
		slog.Float64("score", x),
	)
	return x, c, f, l, n
}

type broadcasterAffectionKey struct{}

var affections = pick.New([]pick.Case[string]{
	{E: "about %[1]f %[2]s", W: 5},
	{E: "roughly %[1]f %[2]s", W: 5},
	{E: "%[1]f or so %[2]s", W: 5},
	{E: "approximately %[1]f %[2]s", W: 5},
	{E: "I have calculated my affection for you to be exactly %[1]f %[2]s", W: 1},
	{E: "Right now, I'd say %[1]f. But who knows what the future may hold? %[2]s", W: 1},
	{E: "%[1]f, and yes, that is a threat. %[2]s", W: 1},
	{E: "%[1]f, given score = c²/(f+1) + (c+1)f + √l, f=%[4]d from your messages sent, l=%[5]d from length of your longest message, and c=%[3]d from memes, across %[6]d messages in the last fifteen minutes %[2]s", W: 1},
})

// Affection describes the caller's affection MMR.
// No arguments.
func Affection(ctx context.Context, robo *Robot, call *Invocation) {
	x, c, f, l, n := score(robo.Log, &call.Channel.History, call.Message.Sender)
	// Anything we do will require an emote.
	e := call.Channel.Emotes.Pick(rand.Uint32())
	if x == 0 {
		// Check for the broadcaster. They get special treatment.
		if strings.EqualFold(call.Message.Name, strings.TrimPrefix(call.Channel.Name, "#")) {
			if _, ok := call.Channel.Extra.LoadOrStore(broadcasterAffectionKey{}, struct{}{}); ok {
				call.Channel.Message(ctx, message.Format("", "Don't make me repeat myself, it's embarrassing! %s", e).AsReply(call.Message.ID))
				return
			}
			const funnyMessage = `It's a bit awkward to think of you like that, streamer... But, well, it's so fun to be here, and I have you to thank for that! So I'd say a whole bunch! %s`
			call.Channel.Message(ctx, message.Format("", funnyMessage, e).AsReply(call.Message.ID))
			return
		}
		// possible!
		call.Channel.Message(ctx, message.Format("", "literally zero %s", e).AsReply(call.Message.ID))
		return
	}
	s := affections.Pick(rand.Uint32())
	// The single scenario where message.Format requiring a constant formatting
	// string is a drawback:
	call.Channel.Message(ctx, message.Sent{Reply: call.Message.ID, Text: fmt.Sprintf(s, x, e, c, f, l, n)})
}

type partnerKey struct{}

type partner struct {
	who   string
	until time.Time
}

// Marry proposes to the robo.
//   - partnership: Type of partnership requested, e.g. "wife", "waifu", "daddy". Optional.
func Marry(ctx context.Context, robo *Robot, call *Invocation) {
	x, _, _, _, _ := score(robo.Log, &call.Channel.History, call.Message.Sender)
	e := call.Channel.Emotes.Pick(rand.Uint32())
	broadcaster := strings.EqualFold(call.Message.Name, strings.TrimPrefix(call.Channel.Name, "#")) && x == 0
	if x < 10 && !broadcaster {
		call.Channel.Message(ctx, message.Format("", "no %s", e).AsReply(call.Message.ID))
		return
	}
	me := &partner{who: call.Message.Sender, until: call.Message.Time().Add(time.Hour)}
	for {
		l, ok := call.Channel.Extra.LoadOrStore(partnerKey{}, me)
		if !ok {
			// No competition. We're a shoo-in.
			call.Channel.Message(ctx, message.Format("", "sure why not %s", e).AsReply(call.Message.ID))
			return
		}
		cur := l.(*partner)
		if cur.who == me.who {
			if rand.Uint32() <= 0xffffffff*3/5 {
				if !call.Channel.Extra.CompareAndDelete(partnerKey{}, cur) {
					// Partner changed concurrently.
					// Really we are guaranteed to fail on time now,
					// but start over anyway.
					continue
				}
				call.Channel.Message(ctx, message.Format("", "How could you forget we're already together? I hate you! Unsubbed, unfollowed, unloved! %s", e).AsReply(call.Message.ID))
				return
			}
			call.Channel.Message(ctx, message.Format("", "We're already together, silly! You're so funny and cute haha. %s", e).AsReply(call.Message.ID))
			return
		}
		if call.Message.Time().Before(cur.until) {
			call.Channel.Message(ctx, message.Format("", "My heart yet belongs to another... %s", e).AsReply(call.Message.ID))
			return
		}
		y, _, _, _, _ := score(robo.Log, &call.Channel.History, cur.who)
		if x < y && !broadcaster {
			call.Channel.Message(ctx, message.Format("", "I'm touched, but I must decline. I'm in love with someone else. %s", e).AsReply(call.Message.ID))
			return
		}
		if !call.Channel.Extra.CompareAndSwap(partnerKey{}, cur, me) {
			// Partner changed concurrently.
			continue
		}
		// We win. Now just decide which message to send.
		// TODO(zeph): since pick.Dist exists now, we could randomize
		if call.Args["partnership"] != "" {
			call.Channel.Message(ctx, message.Format("", "Yes! I'll be your %s! %s", call.Args["partnership"], e).AsReply(call.Message.ID))
		} else {
			call.Channel.Message(ctx, message.Format("", "Yes! I'll marry you! %s", e).AsReply(call.Message.ID))
		}
		return
	}
}

// DescribeMarriage gives some exposition about the marriage system.
// No args.
func DescribeMarriage(ctx context.Context, robo *Robot, call *Invocation) {
	const s = `I am looking for a long series of short-term relationships and am holding a ranked competitive how-much-I-like-you tournament to decide my suitors! Politely ask me to marry you (or become your partner) and I'll evaluate your score. I like copypasta, memes, and long walks in the chat.`
	call.Channel.Message(ctx, message.Sent{Text: s})
}
