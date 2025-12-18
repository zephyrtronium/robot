package message

import (
	"strconv"
	"strings"

	"gitlab.com/zephyrtronium/tmi"
)

// FromTMI adapts a TMI IRC message.
func FromTMI(m *tmi.Message) *Received[User] {
	id, _ := m.Tag("id")
	sender, _ := m.Tag("user-id")
	ts, _ := m.Tag("tmi-sent-ts")
	u, _ := strconv.ParseInt(ts, 10, 64)
	r := Received[User]{
		ID:          id,
		To:          m.To(),
		Sender:      User{ID: sender, Name: m.DisplayName()},
		Text:        m.Trailing,
		Timestamp:   u,
		IsModerator: moderator(m),
		IsElevated:  elevated(m),
	}
	return &r
}

func moderator(m *tmi.Message) bool {
	// The mod tag is unreliable, as it is false for broadcasters and
	// lead moderators. Badges are the only reliable source for this info.
	badges, _ := m.Tag("badges")
	for badges != "" {
		b, rest, _ := strings.Cut(badges, ",")
		b, _, _ = strings.Cut(b, "/")
		switch b {
		case "broadcaster", "lead_moderator", "moderator":
			return true
		}
		badges = rest
	}
	return false
}

func elevated(m *tmi.Message) bool {
	sub, _ := m.Tag("subscriber")
	if sub == "1" {
		return true
	}
	_, vip := m.Tag("vip")
	// Fortunately, Twitch documentation no longer demands checking badges for this info.
	return vip
}

// ToTMI creates a message to send to TMI. If reply is not empty, then the
// result is a reply to the message with that ID.
func ToTMI(msg Sent) *tmi.Message {
	r := tmi.Privmsg(msg.To, msg.Text)
	if msg.Reply != "" {
		r.Tags = "reply-parent-msg-id=" + msg.Reply
	}
	return r
}
