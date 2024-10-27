package command

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/zephyrtronium/robot/channel"
	"gitlab.com/zephyrtronium/pick"
)

func score(log *slog.Logger, h *channel.History, user string) (x float64, c, f, l, n int) {
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
	{E: "%.0[1]f %.1[1]f %.2[1]f %.3[1]f %.4[1]f %.5[1]f %.6[1]f %.7[1]f %.8[1]f %.9[1]f", W: 1},
	{E: "%[1]f, given score = c²/(f+1) + (c+1)f + √l, f=%[4]d from your messages sent, l=%[5]d from length of your longest message, and c=%[3]d from memes, across %[6]d messages in the last fifteen minutes %[2]s", W: 1},
})

// Affection describes the caller's affection MMR.
// No arguments.
func Affection(ctx context.Context, robo *Robot, call *Invocation) {
	x, c, f, l, n := score(robo.Log, call.Channel.History, call.Message.Sender)
	// Anything we do will require an emote.
	e := call.Channel.Emotes.Pick(rand.Uint32())
	if x == 0 {
		// Check for the broadcaster. They get special treatment.
		if strings.EqualFold(call.Message.Name, strings.TrimPrefix(call.Channel.Name, "#")) {
			if _, ok := call.Channel.Extra.LoadOrStore(broadcasterAffectionKey{}, struct{}{}); ok {
				call.Channel.Message(ctx, call.Message.ID, "Don't make me repeat myself, it's embarrassing! "+e)
				return
			}
			const funnyMessage = `It's a bit awkward to think of you like that, streamer... But, well, it's so fun to be here, and I have you to thank for that! So I'd say a whole bunch!`
			call.Channel.Message(ctx, call.Message.ID, funnyMessage+" "+e)
			return
		}
		// possible!
		call.Channel.Message(ctx, call.Message.ID, "literally zero "+e)
		return
	}
	s := affections.Pick(rand.Uint32())
	call.Channel.Message(ctx, call.Message.ID, fmt.Sprintf(s, x, e, c, f, l, n))
}

type partnerKey struct{}

type partner struct {
	who   string
	until time.Time
}

// Marry proposes to the robo.
//   - partnership: Type of partnership requested, e.g. "wife", "waifu", "daddy". Optional.
func Marry(ctx context.Context, robo *Robot, call *Invocation) {
	x, _, _, _, _ := score(robo.Log, call.Channel.History, call.Message.Sender)
	e := call.Channel.Emotes.Pick(rand.Uint32())
	broadcaster := strings.EqualFold(call.Message.Name, strings.TrimPrefix(call.Channel.Name, "#")) && x == 0
	if x < 10 && !broadcaster {
		call.Channel.Message(ctx, call.Message.ID, "no "+e)
		return
	}
	me := &partner{who: call.Message.Sender, until: call.Message.Time().Add(time.Hour)}
	for {
		l, ok := call.Channel.Extra.LoadOrStore(partnerKey{}, me)
		if !ok {
			// No competition. We're a shoo-in.
			call.Channel.Message(ctx, call.Message.ID, "sure why not "+e)
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
				call.Channel.Message(ctx, call.Message.ID, "How could you forget we're already together? I hate you! Unsubbed, unfollowed, unloved! "+e)
				return
			}
			call.Channel.Message(ctx, call.Message.ID, "We're already together, silly! You're so funny and cute haha. "+e)
			return
		}
		if call.Message.Time().Before(cur.until) {
			call.Channel.Message(ctx, call.Message.ID, "My heart yet belongs to another... "+e)
			return
		}
		y, _, _, _, _ := score(robo.Log, call.Channel.History, cur.who)
		if x < y && !broadcaster {
			call.Channel.Message(ctx, call.Message.ID, "I'm touched, but I must decline. I'm in love with someone else. "+e)
			return
		}
		if !call.Channel.Extra.CompareAndSwap(partnerKey{}, cur, me) {
			// Partner changed concurrently.
			continue
		}
		// We win. Now just decide which message to send.
		// TODO(zeph): since pick.Dist exists now, we could randomize
		if call.Args["partnership"] != "" {
			call.Channel.Message(ctx, call.Message.ID, fmt.Sprintf("Yes! I'll be your %s! %s", call.Args["partnership"], e))
		} else {
			call.Channel.Message(ctx, call.Message.ID, "Yes! I'll marry you! "+e)
		}
		return
	}
}

// DescribeMarriage gives some exposition about the marriage system.
// No args.
func DescribeMarriage(ctx context.Context, robo *Robot, call *Invocation) {
	const s = `I am looking for a long series of short-term relationships and am holding a ranked competitive how-much-I-like-you tournament to decide my suitors! Politely ask me to marry you (or become your partner) and I'll evaluate your score. I like copypasta, memes, and long walks in the chat.`
	call.Channel.Message(ctx, "", s)
}
