package message

import (
	"strconv"

	"gitlab.com/zephyrtronium/tmi"
)

// FromTMI adapts a TMI IRC message.
func FromTMI(m *tmi.Message) *Received {
	id, _ := m.Tag("id")
	sender, _ := m.Tag("user-id")
	ts, _ := m.Tag("tmi-sent-ts")
	u, _ := strconv.ParseInt(ts, 10, 64)
	r := Received{
		ID:          id,
		To:          m.To(),
		Sender:      sender,
		Name:        m.DisplayName(),
		Text:        m.Trailing,
		Timestamp:   u,
		IsModerator: moderator(m),
		IsElevated:  elevated(m),
	}
	return &r
}

func moderator(m *tmi.Message) bool {
	t, _ := m.Tag("mod")
	if t == "1" {
		return true
	}
	// The broadcaster seems to get mod=0, but their nick is equal to the
	// channel name.
	if to := m.To(); to[0] == '#' && to[1:] == m.Nick {
		return true
	}
	// We could additionally check badges and user-type, but that's a lot of
	// scanning tags for not much gain.
	return false
}

func elevated(m *tmi.Message) bool {
	sub, _ := m.Tag("subscriber")
	if sub == "1" {
		return true
	}
	vip, _ := m.Tag("vip")
	// Again, we could check badges, but those tend to be unreliable anyway,
	// not to mention subject to change.
	return vip == "1"
}

// ToTMI creates a message to send to TMI. If reply is not empty, then the
// result is a reply to the message with that ID.
func ToTMI(reply, to, text string) *tmi.Message {
	r := tmi.Privmsg(to, text)
	if reply != "" {
		r.Tags = "reply-parent-msg-id=" + reply
	}
	return r
}
