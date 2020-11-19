package irc_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/zephyrtronium/robot/irc"
)

func TestParse(t *testing.T) {
	cases := []struct {
		text string
		msg  irc.Message
		ok   bool
	}{
		// command only
		{"376\r\n", irc.Message{Command: "376"}, true},
		{"PRIVMSG\r\n", irc.Message{Command: "PRIVMSG"}, true},
		// command with sender
		{":tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}}, true},
		{":madoka@tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}}, true},
		// command with params
		{"PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Params: []string{"#madoka"}}, true},
		{"PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Params: []string{"#madoka", "#homura"}}, true},
		// command with trailing
		{"PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Trailing: "anime"}, true},
		{"PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Trailing: "anime madoka homura"}, true},
		{"PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Trailing: ""}, true},
		// sender, command, params
		{":tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}}, true},
		{":madoka@tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}, Params: []string{"#madoka"}}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}, Params: []string{"#madoka"}}, true},
		{":tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}}, true},
		{":madoka@tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}}, true},
		// sender, command, trailing
		{":tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime"}, true},
		{":madoka@tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}, Trailing: "anime"}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}, Trailing: "anime"}, true},
		{":tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime madoka homura"}, true},
		{":madoka@tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}, Trailing: "anime madoka homura"}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}, Trailing: "anime madoka homura"}, true},
		{":tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: ""}, true},
		{":madoka@tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}, Trailing: ""}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}, Trailing: ""}, true},
		// sender, command, params, trailing
		{":tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime", Params: []string{"#madoka"}}, true},
		{":madoka@tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}, Trailing: "anime", Params: []string{"#madoka"}}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}, Trailing: "anime", Params: []string{"#madoka"}}, true},
		{":tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime madoka homura", Params: []string{"#madoka"}}, true},
		{":madoka@tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}, Trailing: "anime madoka homura", Params: []string{"#madoka"}}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}, Trailing: "anime madoka homura", Params: []string{"#madoka"}}, true},
		{":tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "", Params: []string{"#madoka"}}, true},
		{":madoka@tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}, Trailing: "", Params: []string{"#madoka"}}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}, Trailing: "", Params: []string{"#madoka"}}, true},
		{":tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime", Params: []string{"#madoka", "#homura"}}, true},
		{":madoka@tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}, Trailing: "anime", Params: []string{"#madoka", "#homura"}}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}, Trailing: "anime", Params: []string{"#madoka", "#homura"}}, true},
		{":tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime madoka homura", Params: []string{"#madoka", "#homura"}}, true},
		{":madoka@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}, Trailing: "anime madoka homura", Params: []string{"#madoka", "#homura"}}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}, Trailing: "anime madoka homura", Params: []string{"#madoka", "#homura"}}, true},
		{":tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "", Params: []string{"#madoka", "#homura"}}, true},
		{":madoka@tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", Host: "tmi.twitch.tv"}, Trailing: "", Params: []string{"#madoka", "#homura"}}, true},
		{":madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Sender: irc.Sender{Nick: "madoka", User: "homura", Host: "tmi.twitch.tv"}, Trailing: "", Params: []string{"#madoka", "#homura"}}, true},
		// command with tags
		{"@a=b PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b"}, true},
		{"@a=b;c=d PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d"}, true},
		{"@a;c=d PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d"}, true},
		{"@a=b;c PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c"}, true},
		// tags, sender, command
		{"@a=b :tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}}, true},
		// tags, command, params
		{"@a=b PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Params: []string{"#madoka"}}, true},
		{"@a=b;c=d PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Params: []string{"#madoka"}}, true},
		{"@a;c=d PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Params: []string{"#madoka"}}, true},
		{"@a=b;c PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Params: []string{"#madoka"}}, true},
		{"@a=b PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Params: []string{"#madoka", "#homura"}}, true},
		{"@a=b;c=d PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Params: []string{"#madoka", "#homura"}}, true},
		{"@a;c=d PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Params: []string{"#madoka", "#homura"}}, true},
		{"@a=b;c PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Params: []string{"#madoka", "#homura"}}, true},
		// tags, command, trailing
		{"@a=b PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Trailing: "anime"}, true},
		{"@a=b;c=d PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Trailing: "anime"}, true},
		{"@a;c=d PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Trailing: "anime"}, true},
		{"@a=b;c PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Trailing: "anime"}, true},
		{"@a=b PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Trailing: "anime madoka homura"}, true},
		{"@a=b;c=d PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Trailing: "anime madoka homura"}, true},
		{"@a;c=d PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Trailing: "anime madoka homura"}, true},
		{"@a=b;c PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Trailing: "anime madoka homura"}, true},
		{"@a=b PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Trailing: ""}, true},
		{"@a=b;c=d PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Trailing: ""}, true},
		{"@a;c=d PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Trailing: ""}, true},
		{"@a=b;c PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Trailing: ""}, true},
		// tags, sender, command, params
		{"@a=b :tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}}, true},
		{"@a=b :tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}}, true},
		// tags, sender, command, trailing
		{"@a=b :tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime"}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime"}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime"}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime"}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: "anime"}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: "anime"}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: "anime"}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: "anime"}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: "anime"}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: "anime"}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: "anime"}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: "anime"}, true},
		{"@a=b :tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime madoka homura"}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: "anime madoka homura"}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: "anime madoka homura"}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b :tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: ""}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: ""}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: ""}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Trailing: ""}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: ""}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: ""}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: ""}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Trailing: ""}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: ""}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: ""}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: ""}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Trailing: ""}, true},
		// tags, sender, command, params, trailing
		{"@a=b :tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: "anime"}, true},
		{"@a=b :tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, true},
		{"@a=b :tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, true},
		{"@a=b :tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, true},
		{"@a=b :tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka"}, Trailing: ""}, true},
		{"@a=b :tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},
		{"@a=b;c=d :tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},
		{"@a;c=d :tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},
		{"@a=b;c :tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Nick: "tmi.twitch.tv"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},
		{"@a=b :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},
		{"@a=b;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},
		{"@a;c=d :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},
		{"@a=b;c :madoka@tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},
		{"@a=b :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},
		{"@a=b;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},
		{"@a;c=d :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a;c=d", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},
		{"@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: ""}, true},

		// misc ok messages
		// variant eol
		{"PRIVMSG\n", irc.Message{Command: "PRIVMSG"}, true},
		// many spaces (I accidentally deleted anime at the start of trailing here, but that's ok)
		{"@a=b;c     :madoka!homura@tmi.twitch.tv     PRIVMSG     #madoka     #homura     :     madoka     homura\r\n", irc.Message{Command: "PRIVMSG", Tags: "a=b;c", Sender: irc.Sender{Host: "tmi.twitch.tv", Nick: "madoka", User: "homura"}, Params: []string{"#madoka", "#homura"}, Trailing: "     madoka     homura"}, true},
		// multiple lines (only get the first)
		{"MADOKA\r\nHOMURA\r\n", irc.Message{Command: "MADOKA"}, true},

		// bad forms
		// bad eol
		{text: "PRIVMSG\r"},
		{text: "PRIVMSG\rPRIVMSG"},
		// nul characters
		{text: "\x00@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@\x00a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a\x00=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=\x00b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b\x00;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;\x00c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c\x00 :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c \x00:madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :\x00madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :m\x00adoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :ma\x00doka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :mad\x00oka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :mado\x00ka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madok\x00a!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka\x00!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!\x00homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!h\x00omura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!ho\x00mura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!hom\x00ura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homu\x00ra@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homur\x00a@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura\x00@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@\x00tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@t\x00mi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tm\x00i.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi\x00.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.\x00twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.t\x00witch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.tw\x00itch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twi\x00tch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twit\x00ch.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitc\x00h.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch\x00.tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.\x00tv PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.t\x00v PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv\x00 PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv \x00PRIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv P\x00RIVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PR\x00IVMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRI\x00VMSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIV\x00MSG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVM\x00SG #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMS\x00G #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG\x00 #madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG \x00#madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #\x00madoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #m\x00adoka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #ma\x00doka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #mad\x00oka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #mado\x00ka #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madok\x00a #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka\x00 #homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka \x00#homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #\x00homura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #h\x00omura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #ho\x00mura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #hom\x00ura :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homu\x00ra :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homur\x00a :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura\x00 :anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura \x00:anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :\x00anime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :a\x00nime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :an\x00ime madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :ani\x00me madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anim\x00e madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime\x00 madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime \x00madoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime m\x00adoka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime ma\x00doka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime mad\x00oka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime mado\x00ka homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madok\x00a homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka\x00 homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka \x00homura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka h\x00omura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka ho\x00mura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka hom\x00ura\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homu\x00ra\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homur\x00a\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\x00\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv PRIVMSG #madoka #homura :anime madoka homura\r\x00\n"},
		// tags and/or sender only
		{text: ":madoka\r\n"},
		{text: ":madoka@tmi.twitch.tv\r\n"},
		{text: ":madoka!homura@tmi.twitch.tv\r\n"},
		{text: "@a=b;c\r\n"},
		{text: "@a=b;c :madoka\r\n"},
		{text: "@a=b;c :madoka@tmi.twitch.tv\r\n"},
		{text: "@a=b;c :madoka!homura@tmi.twitch.tv\r\n"},

		// real-world failures
		{
			text: "@badge-info=subscriber/3;badges=subscriber/3;client-nonce=00000000000000000000000000000000;color=#00A651;display-name=xxxxxxxx;emotes=;flags=;id=00000000-0000-0000-0000-000000000000;mod=0;room-id=xxxxxxxx;subscriber=1;tmi-sent-ts=0000000000000;turbo=0;user-id=00000000;user-type= :xxxxxxxx!xxxxxxxx@xxxxxxxx.tmi.twitch.tv PRIVMSG #xxxxxxxxx :⣿⣿⣷⡁⢆⠈⠕⢕⢂⢕⢂⢕⢂⢔⢂⢕⢄⠂⣂⠂⠆⢂⢕⢂⢕⢂⢕⢂⢕⢂ ⣿⣿⣿⡷⠊⡢⡹⣦⡑⢂⢕⢂⢕⢂⢕⢂⠕⠔⠌⠝⠛⠶⠶⢶⣦⣄⢂⢕⢂⢕ ⣿⣿⠏⣠⣾⣦⡐⢌⢿⣷⣦⣅⡑⠕⠡⠐⢿⠿⣛⠟⠛⠛⠛⠛⠡⢷⡈⢂⢕⢂ ⠟⣡⣾⣿⣿⣿⣿⣦⣑⠝⢿⣿⣿⣿⣿⣿⡵⢁⣤⣶⣶⣿⢿⢿⢿⡟⢻⣤⢑⢂ ⣾⣿⣿⡿⢟⣛⣻⣿⣿⣿⣦⣬⣙⣻⣿⣿⣷⣿⣿⢟⢝⢕⢕⢕⢕⢽⣿⣿⣷⣔ ⣿⣿⠵⠚⠉⢀⣀⣀⣈⣿⣿⣿⣿⣿⣿⣿⣿⣿⣗⢕⢕⢕⢕⢕⢕⣽⣿⣿⣿⣿ ⢷⣂⣠⣴⣾⡿⡿⡻⡻⣿⣿⣴⣿⣿⣿⣿⣿⣿⣷⣵⣵⣵⣷⣿⣿⣿⣿⣿⣿⡿ ⢌⠻⣿⡿⡫⡪⡪⡪⡪⣺⣿⣿⣿⣿⣿⠿⠿⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠃ ⠣⡁⠹⡪⡪⡪⡪⣪⣾⣿⣿⣿⣿⠋⠐⢉⢍⢄⢌⠻⣿⣿⣿⣿⣿⣿⣿⣿⠏⠈ ⡣⡘⢄⠙⣾⣾⣾⣿⣿⣿⣿⣿⣿⡀⢐⢕⢕⢕⢕⢕⡘⣿⣿⣿⣿⣿⣿⠏⠠⠈ ⠌⢊⢂⢣⠹⣿⣿⣿⣿⣿⣿⣿⣿⣧⢐⢕⢕⢕⢕⢕⢅⣿⣿⣿⣿⡿⢋⢜⠠⠈ ⠄⠁⠕⢝⡢⠈⠻⣿⣿⣿⣿⣿⣿⣿⣷⣕⣑⣑⣑⣵⣿⣿⣿⡿⢋⢔⢕⣿⠠⠈ ⠨⡂⡀⢑⢕⡅⠂⠄⠉⠛⠻⠿⢿⣿⣿⣿⣿⣿⣿⣿⣿⡿⢋⢔⢕⢕⣿⣿⠠⠈ ⠄⠪⣂⠁⢕⠆⠄⠂⠄⠁⡀⠂⡀⠄⢈⠉⢍⢛⢛⢛⢋⢔⢕⢕⢕⣽⣿⣿⠠⠈\r\n",
			msg: irc.Message{
				Command:  "PRIVMSG",
				Tags:     "badge-info=subscriber/3;badges=subscriber/3;client-nonce=00000000000000000000000000000000;color=#00A651;display-name=xxxxxxxx;emotes=;flags=;id=00000000-0000-0000-0000-000000000000;mod=0;room-id=xxxxxxxx;subscriber=1;tmi-sent-ts=0000000000000;turbo=0;user-id=00000000;user-type=",
				Sender:   irc.Sender{Nick: "xxxxxxxx", User: "xxxxxxxx", Host: "xxxxxxxx.tmi.twitch.tv"},
				Params:   []string{"#xxxxxxxxx"},
				Trailing: "⣿⣿⣷⡁⢆⠈⠕⢕⢂⢕⢂⢕⢂⢔⢂⢕⢄⠂⣂⠂⠆⢂⢕⢂⢕⢂⢕⢂⢕⢂ ⣿⣿⣿⡷⠊⡢⡹⣦⡑⢂⢕⢂⢕⢂⢕⢂⠕⠔⠌⠝⠛⠶⠶⢶⣦⣄⢂⢕⢂⢕ ⣿⣿⠏⣠⣾⣦⡐⢌⢿⣷⣦⣅⡑⠕⠡⠐⢿⠿⣛⠟⠛⠛⠛⠛⠡⢷⡈⢂⢕⢂ ⠟⣡⣾⣿⣿⣿⣿⣦⣑⠝⢿⣿⣿⣿⣿⣿⡵⢁⣤⣶⣶⣿⢿⢿⢿⡟⢻⣤⢑⢂ ⣾⣿⣿⡿⢟⣛⣻⣿⣿⣿⣦⣬⣙⣻⣿⣿⣷⣿⣿⢟⢝⢕⢕⢕⢕⢽⣿⣿⣷⣔ ⣿⣿⠵⠚⠉⢀⣀⣀⣈⣿⣿⣿⣿⣿⣿⣿⣿⣿⣗⢕⢕⢕⢕⢕⢕⣽⣿⣿⣿⣿ ⢷⣂⣠⣴⣾⡿⡿⡻⡻⣿⣿⣴⣿⣿⣿⣿⣿⣿⣷⣵⣵⣵⣷⣿⣿⣿⣿⣿⣿⡿ ⢌⠻⣿⡿⡫⡪⡪⡪⡪⣺⣿⣿⣿⣿⣿⠿⠿⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠃ ⠣⡁⠹⡪⡪⡪⡪⣪⣾⣿⣿⣿⣿⠋⠐⢉⢍⢄⢌⠻⣿⣿⣿⣿⣿⣿⣿⣿⠏⠈ ⡣⡘⢄⠙⣾⣾⣾⣿⣿⣿⣿⣿⣿⡀⢐⢕⢕⢕⢕⢕⡘⣿⣿⣿⣿⣿⣿⠏⠠⠈ ⠌⢊⢂⢣⠹⣿⣿⣿⣿⣿⣿⣿⣿⣧⢐⢕⢕⢕⢕⢕⢅⣿⣿⣿⣿⡿⢋⢜⠠⠈ ⠄⠁⠕⢝⡢⠈⠻⣿⣿⣿⣿⣿⣿⣿⣷⣕⣑⣑⣑⣵⣿⣿⣿⡿⢋⢔⢕⣿⠠⠈ ⠨⡂⡀⢑⢕⡅⠂⠄⠉⠛⠻⠿⢿⣿⣿⣿⣿⣿⣿⣿⣿⡿⢋⢔⢕⢕⣿⣿⠠⠈ ⠄⠪⣂⠁⢕⠆⠄⠂⠄⠁⡀⠂⡀⠄⢈⠉⢍⢛⢛⢛⢋⢔⢕⢕⢕⣽⣿⣿⠠⠈",
			},
			ok: true,
		},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			r := strings.NewReader(c.text)
			m, err := irc.Parse(r)
			if err != nil {
				if c.ok {
					t.Errorf("error parsing %q: %v", c.text, err)
				}
				return
			} else if !c.ok {
				t.Errorf("expected error parsing %q but got none", c.text)
			}
			// Always use the zero time.
			m.Time = time.Time{}
			if diff := cmp.Diff(c.msg, m); diff != "" {
				t.Errorf("wrong parse (-want +got):\n%s", diff)
			}
		})
	}
}

func TestString(t *testing.T) {
	// We're only really interested in correct formatting of messages we send,
	// so we don't need to test tags or sender.
	cases := []struct {
		in  irc.Message
		out string
	}{
		{irc.Message{Command: "PRIVMSG"}, "PRIVMSG"},
		{irc.Message{Command: "PRIVMSG", Params: []string{"#madoka"}}, "PRIVMSG #madoka"},
		{irc.Message{Command: "PRIVMSG", Params: []string{"#madoka", "#homura"}}, "PRIVMSG #madoka #homura"},
		{irc.Message{Command: "PRIVMSG", Trailing: "anime"}, "PRIVMSG :anime"},
		{irc.Message{Command: "PRIVMSG", Trailing: "anime madoka homura"}, "PRIVMSG :anime madoka homura"},
		{irc.Message{Command: "PRIVMSG", Params: []string{"#madoka"}, Trailing: "anime"}, "PRIVMSG #madoka :anime"},
		{irc.Message{Command: "PRIVMSG", Params: []string{"#madoka", "#homura"}, Trailing: "anime"}, "PRIVMSG #madoka #homura :anime"},
		{irc.Message{Command: "PRIVMSG", Params: []string{"#madoka"}, Trailing: "anime madoka homura"}, "PRIVMSG #madoka :anime madoka homura"},
		{irc.Message{Command: "PRIVMSG", Params: []string{"#madoka", "#homura"}, Trailing: "anime madoka homura"}, "PRIVMSG #madoka #homura :anime madoka homura"},
	}
	for _, c := range cases {
		t.Run(c.out, func(t *testing.T) {
			s := c.in.String()
			if s != c.out {
				t.Errorf("wrong message: expected %q, got %q", c.out, s)
			}
		})
	}
}

func TestTags(t *testing.T) {
	type try struct {
		tag string
		val string
		ok  bool
	}
	cases := []struct {
		tags string
		try  []try
	}{
		{`a`, []try{
			{"a", "", true},
			{"b", "", false},
			{"", "", false},
		}},
		{`a=`, []try{
			{"a", "", true},
			{"b", "", false},
			{"", "", false},
		}},
		{`a=b`, []try{
			{"a", "b", true},
			{"b", "", false},
			{"", "", false},
		}},
		{`a=\:\s\\\r\n\t\x00`, []try{
			{"a", "; \\\r\ntx00", true},
			{"b", "", false},
			{"t", "", false},
			{"x", "", false},
			{"0", "", false},
			{"", "", false},
		}},
		{`a=\`, []try{
			{"a", "", true},
			{"b", "", false},
			{"", "", false},
		}},
		{`a;c`, []try{
			{"a", "", true},
			{"c", "", true},
			{"", "", false},
		}},
		{`a=;c`, []try{
			{"a", "", true},
			{"c", "", true},
			{"", "", false},
		}},
		{`a;c=`, []try{
			{"a", "", true},
			{"c", "", true},
			{"", "", false},
		}},
		{`a=;c=`, []try{
			{"a", "", true},
			{"c", "", true},
			{"", "", false},
		}},
		{`a=b;c`, []try{
			{"a", "b", true},
			{"b", "", false},
			{"c", "", true},
			{"", "", false},
		}},
		{`a=b;c=`, []try{
			{"a", "b", true},
			{"b", "", false},
			{"c", "", true},
			{"", "", false},
		}},
		{`a;c=d`, []try{
			{"a", "", true},
			{"b", "", false},
			{"c", "d", true},
			{"", "", false},
		}},
		{`a=;c=d`, []try{
			{"a", "", true},
			{"b", "", false},
			{"c", "d", true},
			{"", "", false},
		}},
		{`a=b;c=d`, []try{
			{"a", "b", true},
			{"b", "", false},
			{"c", "d", true},
			{"", "", false},
		}},
		{`a=\:\s\\\r\n\t\x00;c=d`, []try{
			{"a", "; \\\r\ntx00", true},
			{"b", "", false},
			{"t", "", false},
			{"x", "", false},
			{"0", "", false},
			{"c", "d", true},
			{"", "", false},
		}},
		{`a=\;c=d`, []try{
			{"a", "", true},
			{"b", "", false},
			{"c", "d", true},
			{"", "", false},
		}},
	}
	for _, c := range cases {
		t.Run(c.tags, func(t *testing.T) {
			m := irc.Message{Tags: c.tags, Command: "PRIVMSG"}
			for _, c := range c.try {
				t.Run(c.tag, func(t *testing.T) {
					r, ok := m.Tag(c.tag)
					if ok != c.ok {
						t.Errorf("tag parse success mismatch, expected %v, got %v", c.ok, ok)
					}
					if r != c.val {
						t.Errorf("tag value mismatch, expected %q, got %q", c.val, r)
					}
				})
			}
		})
	}
}

func TestBadges(t *testing.T) {
	cases := []struct {
		b string
		r []string
	}{
		{"", nil},
		{"a/0", []string{"a"}},
		{"a/0,b/0", []string{"a", "b"}},
	}
	for _, c := range cases {
		t.Run(c.b, func(t *testing.T) {
			m := irc.Message{Tags: "badges=" + c.b, Command: "PRIVMSG"}
			r := m.Badges(nil)
			if diff := cmp.Diff(c.r, r); diff != "" {
				t.Errorf("wrong badges (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDisplayName(t *testing.T) {
	cases := []struct {
		name string
		msg  irc.Message
		n    string
	}{
		{"none", irc.Message{Tags: "", Sender: irc.Sender{Nick: "nick"}}, "nick"},
		{"empty", irc.Message{Tags: "display-name=", Sender: irc.Sender{Nick: "nick"}}, "nick"},
		{"provided", irc.Message{Tags: "display-name=NICK", Sender: irc.Sender{Nick: "nick"}}, "NICK"},
	}
	for _, c := range cases {
		t.Run(c.msg.Tags, func(t *testing.T) {
			if got := c.msg.DisplayName(); got != c.n {
				t.Errorf("wrong display name: want %q, got %q", c.n, got)
			}
		})
	}
}
