package command

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"time"

	"github.com/zephyrtronium/robot/channel"
)

func score(h *channel.History, user string) float64 {
	mine := make(map[string]map[string]struct{})
	f, c, l, n := 0, 0, 0, 0
	for m := range h.All() {
		who, text := m.Sender, m.Text
		n++
		// Count the number of distinct other users who said each of your messages
		// after you said it.
		m, ok := mine[text]
		if who != user {
			if ok {
				slog.Debug("scoring meme", slog.String("user", user), slog.String("memer", who), slog.String("msg", text))
				m[who] = struct{}{}
				c += len(m) // n.b. quadratic growth for this component
			}
			continue
		}
		// Count the messages you sent.
		if !ok {
			slog.Debug("scoring first", slog.String("user", user), slog.String("msg", text))
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
	x := float64(c*c)/float64(f+1) + float64(c*f+f) + math.Sqrt(float64(l))
	slog.Info("user score",
		slog.String("user", user),
		slog.Int("msgs", n),
		slog.Int("freq", f),
		slog.Int("meme", c),
		slog.Int("len", l),
		slog.Float64("score", x),
	)
	return x
}

// Affection describes the caller's affection MMR.
// No arguments.
func Affection(ctx context.Context, robo *Robot, call *Invocation) {
	x := score(call.Channel.History, call.Message.Sender)
	// Anything we do will require an emote.
	e := call.Channel.Emotes.Pick(rand.Uint32())
	if x == 0 {
		// possible!
		call.Channel.Message(ctx, call.Message.ID, "literally zero "+e)
		return
	}
	call.Channel.Message(ctx, call.Message.ID, fmt.Sprintf("about %f %s", x, e))
}

type partnerKey struct{}

type partner struct {
	who   string
	until time.Time
}

// Marry proposes to the robo.
//   - partnership: Type of partnership requested, e.g. "wife", "waifu", "daddy". Optional.
func Marry(ctx context.Context, robo *Robot, call *Invocation) {
	x := score(call.Channel.History, call.Message.Sender)
	e := call.Channel.Emotes.Pick(rand.Uint32())
	if x < 10 {
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
		y := score(call.Channel.History, cur.who)
		if x < y {
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
