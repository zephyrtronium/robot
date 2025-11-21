package channel

import (
	"context"
	"errors"
	"regexp"
	"strconv"
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
	// Links, BotCommands, and OneWord control handling of messages that contain
	// apparent links, commands for other bots, and other messages not containing
	// whitespace, respectively.
	Links, BotCommands, OneWord BlockOption
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
	// Permissions is the set of users with special permissions granted through
	// static config.
	Permissions map[string]UserPerms
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

// UserPerms is the permissions for a user granted by static configuration.
type UserPerms struct {
	// DisableCommands means the user has access to no commands.
	// This overrides moderator status.
	DisableCommands bool
	// DisableLearn means the bot never learns the user's messages.
	// This is independent of the privacy list.
	DisableLearn bool
	// DisableSpeak means the user's messages won't trigger the bot to
	// speak unprompted.
	DisableSpeak bool
	// DisableMemes means the user's messages don't count toward copypasta.
	DisableMemes bool
	// Moderator means the user has access to moderator commands if not
	// disabled by DisableCommands.
	Moderator bool
}

// BlockOption is the strategy for handling a variety of message.
type BlockOption int

const (
	// DefaultBlock causes channel config to fall back to the global setting.
	// It is illegal for it to appear in global config.
	DefaultBlock BlockOption = iota
	// Block causes the bot to ignore the message.
	Block
	// Meme allows the bot to copypasta the message, but prevents learning.
	Meme
	// Learn allows the bot to learn from the message.
	Learn
	// NOTE(zephyr): Message handling assumes that smaller values are more restrictive.
)

func (b BlockOption) String() string {
	switch b {
	case DefaultBlock:
		return ""
	case Block:
		return "block"
	case Meme:
		return "meme"
	case Learn:
		return "learn"
	default:
		return "BlockOption(" + strconv.Itoa(int(b)) + ")"
	}
}

func (b *BlockOption) UnmarshalText(text []byte) error {
	switch string(text) {
	case "":
		*b = DefaultBlock
	case "block":
		*b = Block
	case "meme":
		*b = Meme
	case "learn":
		*b = Learn
	default:
		return errors.New(`invalid block option ` + strconv.Quote(string(text)) + `; must be "block", "meme", or "learn" if specified`)
	}
	return nil
}

func (b *BlockOption) MarshalText() ([]byte, error) {
	if b == nil {
		return nil, nil
	}
	switch *b {
	case DefaultBlock:
		return nil, nil
	case Block:
		return []byte("block"), nil
	case Meme:
		return []byte("meme"), nil
	case Learn:
		return []byte("learn"), nil
	default:
		return nil, errors.New("impossible block option " + b.String())
	}
}
