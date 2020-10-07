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

package irc

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Message represents a single Twitch IRC Message.
type Message struct {
	Time time.Time
	Tags string
	Sender
	Command  string
	Params   []string
	Trailing string
}

// Privmsg is a shortcut to create a PRIVMSG message for sending.
func Privmsg(to, message string) Message {
	return Message{Command: "PRIVMSG", Params: []string{to}, Trailing: message}
}

// Whisper is a shortcut to create a Twitch whisper to a given user.
func Whisper(to, message string) Message {
	return Message{
		Command:  "PRIVMSG",
		Params:   []string{"#jtv"},
		Trailing: "/w " + to + " " + message,
	}
}

// Reply creates an appropriately-targeted response to a PRIVMSG or WHISPER.
// Panics if the message type is neither PRIVMSG nor WHISPER. The message is
// formatted according to the rules of fmt.Sprintf.
func (m Message) Reply(format string, args ...interface{}) Message {
	msg := strings.TrimSpace(fmt.Sprintf(format, args...))
	switch m.Command {
	case "PRIVMSG":
		return Privmsg(m.To(), msg)
	case "WHISPER":
		return Whisper(m.Nick, msg)
	default:
		panic("robot/irc: cannot respond to " + m.String())
	}
}

// String formats the message as an IRC message string.
func (m Message) String() string {
	var b strings.Builder
	if len(m.Tags) != 0 {
		b.WriteByte('@')
		b.WriteString(m.Tags)
		b.WriteByte(' ')
	}
	snd := m.Sender.String()
	if snd != "" {
		b.WriteByte(':')
		b.WriteString(snd)
		b.WriteByte(' ')
	}
	b.WriteString(m.Command)
	for _, p := range m.Params {
		b.WriteByte(' ')
		b.WriteString(p)
	}
	if m.Trailing != "" {
		b.WriteByte(' ')
		b.WriteByte(':')
		b.WriteString(m.Trailing)
	}
	return b.String()
}

// Text formats the message as a short display string.
func (m Message) Text() string {
	var b strings.Builder
	if m.Nick != "" {
		b.WriteByte(':')
		b.WriteString(m.Nick)
		b.WriteByte(' ')
	}
	b.WriteString(m.Command)
	for _, p := range m.Params {
		b.WriteByte(' ')
		b.WriteString(p)
	}
	if m.Trailing != "" {
		b.WriteByte(' ')
		b.WriteByte(':')
		b.WriteString(m.Trailing)
	}
	return b.String()
}

// Tag retrieves a tag by name.
func (m Message) Tag(name string) (string, bool) {
	tags := m.Tags
	for tags != "" {
		k := strings.IndexByte(tags, ';')
		tag := tags
		if k >= 0 {
			tag = tags[:k]
			tags = tags[k+1:]
		} else {
			tags = ""
		}
		k = strings.IndexByte(tag, '=')
		var key, val string
		if k >= 0 {
			key = tag[:k]
			val = tag[k+1:]
		} else {
			key = tag
		}
		if key == name {
			return val, true
		}
	}
	return "", false
}

// To returns m.Params[0]. Panics if m.Params is empty.
func (m Message) To() string {
	return m.Params[0]
}

// Sender is a message Sender. It may represent a user or server.
type Sender struct {
	Nick string
	User string
	Host string
}

func (s Sender) String() string {
	if s.Nick != "" {
		return s.Nick + "!" + s.User + "@" + s.Host
	}
	return s.Host
}

// Parse parses a message.
func Parse(scan io.RuneScanner) (msg Message, err error) {
	defer func() { msg.Time = time.Now() }()
	var r rune
	r, _, err = scan.ReadRune()
	// Parse tags.
	if r == '@' {
		msg.Tags, err = scanField(scan, "tags", tagLimit)
		if err != nil {
			return
		}
		r, _, err = scan.ReadRune()
		if err != nil {
			return
		}
		if r != ' ' {
			err = Malformed{stage: "message (only has tags)"}
			return
		}
	} else {
		scan.UnreadRune()
	}
	// Parse sender.
	r, _, err = scan.ReadRune()
	if r == ':' {
		msg.Sender, err = scanSender(scan)
		if err != nil {
			return
		}
	} else {
		scan.UnreadRune()
	}
	// Parse command.
	msg.Command, err = scanField(scan, "command", ircLimit)
	if err != nil {
		return
	}
	r, _, err = scan.ReadRune()
	if err != nil {
		// scanField also unreads the last rune.
		panic("unreachable")
	}
	switch r {
	case ' ': // do nothing
	case '\r':
		r, _, err = scan.ReadRune()
		if err != nil {
			return
		}
		if r != '\n' {
			err = Malformed{stage: "message"}
		}
		return
	case '\n':
		return
	default:
		panic("unreachable")
	}
	// Parse middle args.
	r, _, err = scan.ReadRune()
	if err != nil {
		return
	}
	for r != ':' {
		scan.UnreadRune()
		var arg string
		arg, err = scanField(scan, "middle", ircLimit)
		// If we get an arg, always add it, even if the error is non-nil.
		if arg != "" {
			msg.Params = append(msg.Params, arg)
		}
		if err != nil {
			return
		}
		r, _, err = scan.ReadRune()
		if err != nil {
			return
		}
		switch r {
		case ' ':
			r, _, err = scan.ReadRune()
			if err != nil {
				return
			}
		case '\r':
			r, _, err = scan.ReadRune()
			if err != nil {
				return
			}
			if r != '\n' {
				err = Malformed{stage: "message"}
				return
			}
		case '\n':
			return
		default:
			panic("unreachable")
		}
	}
	// Parse trailing.
	msg.Trailing, err = scanLine(scan, "trailing")
	if err != nil {
		return
	}
	r, _, err = scan.ReadRune()
	if err != nil {
		return
	}
	if r != '\n' {
		err = Malformed{"eol"}
	}
	return
}

// tagLimit is the maximum length of a tag in bytes.
const tagLimit = 8192

// ircLimit is the maximum length of an IRC message, excluding tag, in bytes.
const ircLimit = 512

func unquoteTag(value string) string {
	// We try hard to avoid a copy. We already know the tag value is
	// well-formed because it was successfully parsed, so we can return a
	// substring as long as the string contains no escape sequences.
	for k, r := range value {
		switch r {
		case ';':
			return value[:k]
		case '\\':
			return unescapeTag(value[:k], value[k:])
		}
	}
	return value
}

func unescapeTag(raw, quoted string) string {
	var b strings.Builder
	b.WriteString(raw)
	q := false
	for _, r := range quoted {
		if q {
			// Unescape backslash sequence. The sequences are:
			//	\: -> ';'
			//	\s -> ' '
			//	\\ -> '\'
			//	\r -> CR
			//	\n -> LF
			// Any other sequence causes the backslash to be ignored (without error).
			switch r {
			case ':':
				b.WriteByte(';')
			case 's':
				b.WriteByte(' ')
			case 'r':
				b.WriteByte('\r')
			case 'n':
				b.WriteByte('\n')
			default:
				b.WriteRune(r)
			}
			q = false
			continue
		}
		switch r {
		case ';':
			return b.String()
		case '\\':
			q = true
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func scanSender(scan io.RuneScanner) (s Sender, err error) {
	var (
		b    strings.Builder
		n, c int
		r    rune
	)
	cur := &s.Nick
	for c < ircLimit {
		r, n, err = scan.ReadRune()
		if err != nil {
			return
		}
		c += n
		switch r {
		case '!':
			// nick into user
			s.Nick = b.String()
			b.Reset()
			cur = &s.User
		case '@':
			// nick or user into server
			*cur = b.String()
			b.Reset()
			cur = &s.Host
		case ' ':
			// nick, user, or server into finish
			*cur = b.String()
			if *cur == "" {
				err = Malformed{stage: "sender"}
			}
			return
		case '\r', '\n', '\000':
			err = Malformed{stage: "sender"}
			return
		default:
			b.WriteRune(r)
		}
	}
	err = Malformed{stage: "sender"}
	return
}

// scanField scans a single space-separated field and unreads the last rune.
// Check scan for ' ', '\r', '\n' after.
func scanField(scan io.RuneScanner, stage string, limit int) (field string, err error) {
	var (
		b    strings.Builder
		n, c int
		r    rune
	)
	for c < limit {
		r, n, err = scan.ReadRune()
		if err != nil {
			return b.String(), err
		}
		c += n
		switch r {
		case ' ', '\r', '\n':
			scan.UnreadRune()
			return b.String(), nil
		case '\000':
			return "", Malformed{stage: stage}
		default:
			b.WriteRune(r)
		}
	}
	return "", Malformed{stage: stage}
}

// scanLine scans until the end of the line.
func scanLine(scan io.RuneScanner, stage string) (line string, err error) {
	var (
		b    strings.Builder
		n, c int
		r    rune
	)
	for c < ircLimit {
		r, n, err = scan.ReadRune()
		if err != nil {
			return b.String(), err
		}
		c += n
		switch r {
		case '\n':
			scan.UnreadRune()
			fallthrough
		case '\r':
			return b.String(), nil
		case '\000':
			return "", Malformed{stage: stage}
		default:
			b.WriteRune(r)
		}
	}
	return "", Malformed{stage: stage}
}

// Malformed indicates a malformed IRC message.
type Malformed struct {
	stage string
}

func (err Malformed) Error() string {
	return "malformed " + err.stage
}
