package message_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"gitlab.com/zephyrtronium/tmi"

	"github.com/zephyrtronium/robot/message"
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
		{
			name:   "sub",
			msg:    `@badge-info=subscriber/11;badges=subscriber/9;client-nonce=f479247496129b9c07e1a0371a9f2f70;color=#1E90FF;display-name=aSub;emotes=emotesv2_994bd9ea759349d1aa51dca6acca627e:9-16;first-msg=0;flags=;id=2a9bb533-2837-48d0-8aba-032f844c91f6;mod=0;returning-chatter=0;room-id=12345678;subscriber=1;tmi-sent-ts=1662887850257;turbo=0;user-id=87654321;user-type= :asub!asub@asub.tmi.twitch.tv PRIVMSG #channel :hello, world!`,
			id:     "2a9bb533-2837-48d0-8aba-032f844c91f6",
			to:     "#channel",
			sender: "87654321",
			disp:   "aSub",
			text:   "hello, world!",
			time:   time.UnixMilli(1662887850257),
			mod:    false,
			elev:   true,
		},
		{
			name:   "vip-sub",
			msg:    `@badge-info=subscriber/42;badges=vip/1,subscriber/2036;client-nonce=6a8e0f4fba78abbad4b9c42534cecf6e;color=#0000FF;display-name=aVIP;emotes=emotesv2_6a361d22e95148b3b8fabc886720d5d7:0-9;first-msg=0;flags=;id=d2129ccd-0763-434c-bd00-7354bfe1a781;mod=0;returning-chatter=0;room-id=12345678;subscriber=1;tmi-sent-ts=1662885432414;turbo=0;user-id=87654321;user-type=;vip=1 :avip!avip@avip.tmi.twitch.tv PRIVMSG #channel :hello, world!`,
			to:     "#channel",
			id:     "d2129ccd-0763-434c-bd00-7354bfe1a781",
			sender: "87654321",
			disp:   "aVIP",
			text:   "hello, world!",
			time:   time.UnixMilli(1662885432414),
			mod:    false,
			elev:   true,
		},
		{
			name:   "moderator",
			msg:    `@badge-info=;badges=moderator/1;client-nonce=ba6030bcdfa3b13c602415dfc69b4786;color=#008000;display-name=BarryCarIyon;emotes=;first-msg=0;flags=;id=b51051ff-742c-491d-8069-6248bc786e6f;mod=1;returning-chatter=0;room-id=15185913;subscriber=0;tmi-sent-ts=1766005013352;turbo=0;user-id=794780266;user-type=mod :barrycariyon!barrycariyon@barrycariyon.tmi.twitch.tv PRIVMSG #barrycarlyon :test`,
			to:     "#barrycarlyon",
			id:     "b51051ff-742c-491d-8069-6248bc786e6f",
			sender: "794780266",
			disp:   "BarryCarIyon",
			text:   "test",
			time:   time.UnixMilli(1766005013352),
			mod:    true,
			elev:   false,
		},
		{
			name:   "lead-moderator",
			msg:    `@badge-info=;badges=lead_moderator/1;client-nonce=c766a2456e9b5069b3b1f9fb8afde952;color=#008000;display-name=BarryCarIyon;emotes=;first-msg=0;flags=;id=dc6664c2-d164-4482-a5cd-ea29c74c56ff;mod=0;returning-chatter=0;room-id=15185913;subscriber=0;tmi-sent-ts=1766005010370;turbo=0;user-id=794780266;user-type= :barrycariyon!barrycariyon@barrycariyon.tmi.twitch.tv PRIVMSG #barrycarlyon :test`,
			to:     "#barrycarlyon",
			id:     "dc6664c2-d164-4482-a5cd-ea29c74c56ff",
			sender: "794780266",
			disp:   "BarryCarIyon",
			text:   "test",
			time:   time.UnixMilli(1766005010370),
			mod:    true,
			elev:   false,
		},
		{
			name:   "broadcaster",
			msg:    `@badge-info=subscriber/10;badges=broadcaster/1,subscriber/0,ambassador/1;client-nonce=e1d60f4c8eb116e9d8152d4cd9f9c32d;color=#033700;display-name=BarryCarlyon;emotes=;first-msg=0;flags=;id=4c0d6ca0-46ad-40f3-8a98-69c3a93ae019;mod=0;returning-chatter=0;room-id=15185913;subscriber=1;tmi-sent-ts=1732383803915;turbo=0;user-id=15185913;user-type= :barrycarlyon!barrycarlyon@barrycarlyon.tmi.twitch.tv PRIVMSG #barrycarlyon :asdasd`,
			to:     "#barrycarlyon",
			id:     "4c0d6ca0-46ad-40f3-8a98-69c3a93ae019",
			sender: "15185913",
			disp:   "BarryCarlyon",
			text:   "asdasd",
			time:   time.UnixMilli(1732383803915),
			mod:    true,
			elev:   true,
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
			if got := msg.ID; got != c.id {
				t.Errorf("wrong id: want %q, got %q", c.id, got)
			}
			if got := msg.To; got != c.to {
				t.Errorf("wrong to: want %q, got %q", c.to, got)
			}
			if got := msg.Sender.ID; got != c.sender {
				t.Errorf("wrong sender: want %q, got %q", c.sender, got)
			}
			if got := msg.Sender.Name; got != c.disp {
				t.Errorf("wrong display name: want %q, got %q", c.disp, got)
			}
			if got := msg.Text; got != c.text {
				t.Errorf("wrong text: want %q, got %q", c.text, got)
			}
			if got := msg.Time(); !got.Equal(c.time) {
				t.Errorf("wrong time: want %v, got %v", c.time, got)
			}
			if got := msg.IsModerator; got != c.mod {
				t.Errorf("wrong mod: want %t, got %t", c.mod, got)
			}
			if got := msg.IsElevated; got != c.elev {
				t.Errorf("wrong elev: want %t, got %t", c.elev, got)
			}
		})
	}
}
