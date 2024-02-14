package message

import (
	"time"
)

type Message struct {
	// ID is the unique ID of the message.
	ID string
	// To is the destination of the message. This may be the identifier of a
	// room or channel or the name of a user.
	To string
	// Sender is a unique identifier for the message sender.
	Sender string
	// Name is the display name of a message sender.
	Name string
	// Text is the text of a message.
	Text string
	// Timestamp is the timestamp of a message as milliseconds since the
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

func (m *Message) Time() time.Time {
	return time.UnixMilli(m.Timestamp)
}
