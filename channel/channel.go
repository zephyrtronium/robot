package channel

import (
	"context"
	"regexp"
	"sync/atomic"

	"gitlab.com/zephyrtronium/pick"
	"golang.org/x/time/rate"
)

type Channel struct {
	// Name is the name of the channel.
	Name string
	// Message sends a message to the channel with an optional reply message ID.
	Message func(ctx context.Context, reply, text string)
	// Learn and Send are the channel tags.
	Learn, Send string
	// Block is a regex that matches messages which should not be used for
	// learning.
	Block *regexp.Regexp
	// Responses is the probability that a received message will trigger a
	// random response.
	Responses float64
	// Rate is the rate limiter for messages. Attempts to speak in excess of
	// the rate limit are dropped.
	Rate *rate.Limiter
	// Ignore is the set of ignored user IDs.
	Ignore map[string]bool
	// Mod is the set of designated moderators' user IDs.
	Mod map[string]bool
	// History is a list of recent messages seen in the channel.
	// Note that messages which are forgotten due to moderation are not removed
	// from this list in general.
	History *History
	// Memery is the meme detector for the channel.
	Memery *MemeDetector
	// Emotes is the distribution of emotes.
	Emotes *pick.Dist[string]
	// Effects is the distribution of effects.
	Effects *pick.Dist[string]
	// Enabled indicates whether a channel is allowed to learn messages.
	Enabled atomic.Bool
}
