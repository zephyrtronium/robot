/*
Copyright (C) 2020  Branden J Brown

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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
