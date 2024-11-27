package channel

import (
	"context"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/zephyrtronium/pick"
	"golang.org/x/time/rate"

	"github.com/zephyrtronium/robot/message"
)

type Channel struct {
	// Name is the name of the channel.
	Name string
	// Message sends a message to the channel with an optional reply message ID.
	Message func(ctx context.Context, msg message.Sent)
	// Learn and Send are the channel tags.
	Learn, Send string
	// Block is a regex that matches messages which should not be used for
	// learning or copypasta.
	Block *regexp.Regexp
	// Meme is a regex that matches messages which bypass Block only for copypasta.
	Meme *regexp.Regexp
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
	History History[*message.Received[message.User]]
	// Memery is the meme detector for the channel.
	Memery *MemeDetector
	// Emotes is the distribution of emotes.
	Emotes *pick.Dist[string]
	// Effects is the distribution of effects.
	Effects *pick.Dist[string]
	// Silent is the earliest time that speaking and learning is allowed in the
	// channel as nanoseconds from the Unix epoch.
	Silent atomic.Int64
	// Extra is extra channel data that may be added by commands.
	Extra sync.Map // map[any]any; key is a type
	// Enabled indicates whether a channel is allowed to learn messages.
	Enabled atomic.Bool
}

func (ch *Channel) SilentTime() time.Time {
	return time.Unix(0, ch.Silent.Load())
}
