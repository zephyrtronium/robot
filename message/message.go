package message

import (
	"fmt"
	"strings"
	"time"
)

// Received is a message received from a service.
type Received[U comparable] struct {
	// ID is the unique ID of the message.
	ID string
	// To is the destination of the message. This may be the identifier of a
	// room or channel or the name of a user.
	To string
	// Sender is a unique identifier for the message sender.
	// Whether it remains constant for a given sender depends on the semantics
	// of the type argument.
	Sender U
	// Name is the display name of the message sender.
	Name string
	// Text is the text of the message.
	Text string
	// Timestamp is the timestamp of the message as milliseconds since the
	// Unix epoch.
	Timestamp int64
	// IsModerator indicates whether the sender can moderate the room to which
	// the message was sent.
	IsModerator bool
	// IsElevated indicates whether the message sender is known to have
	// elevated privileges with respect to the bot, for example a subscriber
	// on Twitch. This may not implicitly include moderators.
	IsElevated bool
}

func (m *Received[U]) Time() time.Time {
	return time.UnixMilli(m.Timestamp)
}

// Sent is a message to be sent to a service.
type Sent struct {
	// Reply is a message to reply to. If empty, the message is not interpreted
	// as a reply.
	Reply string
	// To is the channel to whom the message is sent.
	To string
	// Text is the message text.
	Text string
}

// formatString is a type to prevent misuse of format strings passed to [Format].
type formatString string

// Format constructs a message to send from a format string literal and
// formatting arguments.
func Format(reply, to string, f formatString, args ...any) Sent {
	return Sent{
		Reply: reply,
		To:    to,
		Text:  strings.TrimSpace(fmt.Sprintf(string(f), args...)),
	}
}
