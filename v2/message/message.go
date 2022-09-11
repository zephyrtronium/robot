package message

import (
	"time"
)

// Interface represents a received message.
type Interface interface {
	// ID returns the unique ID of the message.
	ID() string
	// To returns the destination of the message. This may be the identifier of
	// a room or channel or the name of a user.
	To() string
	// Sender returns a unique identifier for the message sender.
	Sender() string
	// Name returns the display name of the message sender.
	Name() string
	// Text returns the text of the message.
	Text() string
	// Time returns the time at which the message was received.
	Time() time.Time
	// IsModerator returns whether can moderate the room to which the message
	// was sent.
	IsModerator() bool
	// IsElevated returns whether the message sender is known to have elevated
	// privileges, for example a subscriber on Twitch. This may not implicitly
	// include moderators.
	IsElevated() bool

	// String creates a representation of the message suitable for debugging.
	String() string
}
