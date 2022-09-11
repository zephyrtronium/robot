package message_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/zephyrtronium/robot/v2/message"
	"gitlab.com/zephyrtronium/tmi"
)

func TestFromIRC(t *testing.T) {
	cases := []struct {
		name   string
		msg    string
		id     string
		to     string
		sender string
		disp   string
		text   string
		time   time.Time
		mod    bool
		elev   bool
	}{
		{
			name:   "regular",
			msg:    `@badge-info=;badges=;client-nonce=eb10a5865f1231b6e96d6ae2dbcecdb4;color=#B22222;display-name=Someone;emotes=;first-msg=0;flags=;id=a74eb158-9732-4e6f-9150-2648cdf3c902;mod=0;returning-chatter=0;room-id=12345678;subscriber=0;tmi-sent-ts=1662882968379;turbo=0;user-id=123456789;user-type= :someone!someone@someone.tmi.twitch.tv PRIVMSG #channel :hello, world!`,
			id:     "a74eb158-9732-4e6f-9150-2648cdf3c902",
			to:     "#channel",
			sender: "123456789",
			disp:   "Someone",
			text:   "hello, world!",
			time:   time.UnixMilli(1662882968379),
			mod:    false,
			elev:   false,
		},
		// TODO(zeph): more cases
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tm, err := tmi.Parse(strings.NewReader(c.msg + "\r\n"))
			if err != nil && err != io.EOF {
				panic(err)
			}
			msg := message.FromTMI(tm)
			if got := msg.ID(); got != c.id {
				t.Errorf("wrong id: want %q, got %q", c.id, got)
			}
			if got := msg.To(); got != c.to {
				t.Errorf("wrong to: want %q, got %q", c.to, got)
			}
			if got := msg.Sender(); got != c.sender {
				t.Errorf("wrong sender: want %q, got %q", c.to, got)
			}
			if got := msg.Name(); got != c.disp {
				t.Errorf("wrong display name: want %q, got %q", c.disp, got)
			}
			if got := msg.Text(); got != c.text {
				t.Errorf("wrong text: want %q, got %q", c.text, got)
			}
			if got := msg.Time(); !got.Equal(c.time) {
				t.Errorf("wrong time: want %v, got %v", c.time, got)
			}
			if got := msg.IsModerator(); got != c.mod {
				t.Errorf("wrong mod: want %t, got %t", c.mod, got)
			}
			if got := msg.IsElevated(); got != c.elev {
				t.Errorf("wrong elev: want %t, got %t", c.elev, got)
			}
		})
	}
}
