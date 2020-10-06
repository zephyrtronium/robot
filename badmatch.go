package main

import (
	"github.com/zephyrtronium/robot/irc"
)

// badmatch filters PRIVMSG messages that contain bad words.
//
// Currently the only bad word is "." at the start of the message.
func badmatch(msg irc.Message) bool {
	if len(msg.Trailing) < 2 {
		return true
	}
	return msg.Trailing[0] == '.'
}
