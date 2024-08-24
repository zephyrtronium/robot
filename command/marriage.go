package command

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"

	"github.com/zephyrtronium/robot/channel"
)

func score(h *channel.History, user string) float64 {
	mine := make(map[string]map[string]struct{})
	f, c, l, n := 0, 0, 0, 0
	for who, text := range h.All() {
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
	// Recall that the history is limited to 512 messages, which means having
	// more than a couple dozen of your own messages is already impressive.
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
