package message

import (
	"time"

	"gitlab.com/zephyrtronium/tmi"
)

// irc implements Interface with a TMI message.
type irc struct {
	message *tmi.Message
}

// FromTMI adapts a TMI IRC message.
func FromTMI(m *tmi.Message) Interface {
	return &irc{m}
}

func (m *irc) ID() string {
	t, _ := m.message.Tag("id")
	return t
}

func (m *irc) To() string {
	return m.message.To()
}

func (m *irc) Sender() string {
	t, ok := m.message.Tag("user-id")
	if !ok {
		return m.message.Nick
	}
	return t
}

func (m *irc) Name() string {
	return m.message.DisplayName()
}

func (m *irc) Text() string {
	return m.message.Trailing
}

func (m *irc) Time() time.Time {
	return m.message.Time()
}

func (m *irc) IsModerator() bool {
	t, _ := m.message.Tag("mod")
	if t == "1" {
		return true
	}
	// The broadcaster seems to get mod=0, but their nick is equal to the
	// channel name.
	if to := m.message.To(); to[0] == '#' && to[1:] == m.message.Nick {
		return true
	}
	// We could additionally check badges and user-type, but that's a lot of
	// scanning tags for not much gain.
	return false
}

func (m *irc) IsElevated() bool {
	sub, _ := m.message.Tag("subscriber")
	if sub == "1" {
		return true
	}
	vip, _ := m.message.Tag("vip")
	// Again, we could check badges, but those tend to be unreliable anyway,
	// not to mention subject to change.
	return vip == "1"
}

func (m *irc) String() string {
	return m.message.String()
}
