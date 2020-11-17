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

// Package irc implements basic scanning and formatting of IRC messages.
//
// While package irc is designed for Twitch.TV IRC, it should be able to handle
// any valid RFC 1459 messages plus IRCv3 tags. The focus is on simplicity,
// universality, and performance, rather than convenience. Users should be
// familiar with RFC 1459.
//
// This package does not handle IRC connections. It can parse messages from an
// existing IRC connection via an io.RuneScanner wrapper, such as bufio.Reader.
//
package irc

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Message represents a single Twitch IRC Message.
type Message struct {
	// Time is the time at which the message was sent.
	Time time.Time
	// Tags is the full tags component of the received message. Use the Tag
	// method to get the parsed, unquoted value of a single tag.
	Tags string
	// Sender is identifying information of the user or server that sent the
	// message.
	Sender
	// Command is the message command or numeric response code.
	Command string
	// Params is the "middle" parameters of the message.
	Params []string
	// Trailing is the "trailing" parameter of the message.
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

// String formats the message as an IRC message string appropriate for sending
// to an IRC server, not including the ending CR LF sequence. This does not
// perform any validation.
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

// Tag retrieves a tag by name. ok is false if and only if the tag is not
// present.
func (m Message) Tag(name string) (val string, ok bool) {
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
			return unquoteTag(val), true
		}
	}
	return "", false
}

// Badges appends the list of Twitch badges, parsed from the badges tag and
// without versions, to v and returns v.
func (m Message) Badges(v []string) []string {
	bb, _ := m.Tag("badges")
	for bb != "" {
		// Index rather than use Split to avoid unnecessary allocations.
		k := strings.IndexByte(bb, ',')
		b := bb
		if k >= 0 {
			b = b[:k]
			bb = bb[k+1:]
		} else {
			bb = ""
		}
		k = strings.IndexByte(b, '/')
		// We should always enter this branch, but it isn't worth panicking
		// if we don't.
		if k >= 0 {
			b = b[:k]
		}
		v = append(v, b)
	}
	return v
}

// To returns m.Params[0]. Panics if m.Params is empty.
//
// Notably, this identifies the channel or user a PRIVMSG message is sent to.
func (m Message) To() string {
	return m.Params[0]
}

// Sender is a message Sender. It may represent a user or server.
type Sender struct {
	// Nick is the nickname of the user who produced the message, or the
	// hostname of the server for messages not produced by users.
	Nick string
	// User is the username of the user who produced the message, if any. For
	// Twitch IRC, this is always the same as Nick if it is nonempty.
	User string
	// Host is the hostname of the user who produced the message, if any. For
	// Twitch IRC, this is always "tmi.twitch.tv" or "<user>.tmi.twitch.tv",
	// where <user> is the username of the authenticated client.
	Host string
}

// String formats the sender as "nick!user@host". Separators are omitted for
// empty fields where valid.
func (s Sender) String() string {
	if s.Host != "" {
		if s.User != "" {
			return s.Nick + "!" + s.User + "@" + s.Host
		}
		return s.Nick + "@" + s.Host
	}
	return s.Nick
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
		if err = eatSpace(scan); err != nil {
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
		if err = eatSpace(scan); err != nil {
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
	case ' ':
		if err = eatSpace(scan); err != nil {
			return
		}
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
		}
		return
	case '\n':
		return
	default:
		panic("unreachable")
	}
	// Parse middle args.
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
			if err = eatSpace(scan); err != nil {
				return
			}
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

// tagLimit is the maximum length of a tag in runes.
const tagLimit = 8192

// ircLimit is the maximum length of an IRC message, excluding tag, in runes.
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
		b strings.Builder
		r rune
	)
	cur := &s.Nick
	for i := 0; i < ircLimit; i++ {
		r, _, err = scan.ReadRune()
		if err != nil {
			return
		}
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
		b strings.Builder
		r rune
	)
	for i := 0; i < limit; i++ {
		r, _, err = scan.ReadRune()
		if err != nil {
			return b.String(), err
		}
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
		b strings.Builder
		r rune
	)
	for i := 0; i < ircLimit; i++ {
		r, _, err = scan.ReadRune()
		if err != nil {
			return b.String(), err
		}
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

// eatSpace scans until the next character that is not U+0020 and unreads it.
func eatSpace(scan io.RuneScanner) error {
	var (
		r   rune
		err error
	)
	for i := 0; i < ircLimit; i++ {
		r, _, err = scan.ReadRune()
		if err != nil {
			return err
		}
		switch r {
		case ' ':
			continue
		case '\000':
			return Malformed{stage: "space"}
		default:
			return scan.UnreadRune()
		}
	}
	return Malformed{stage: "space"}
}

// Malformed indicates a malformed IRC message.
type Malformed struct {
	stage string
}

func (err Malformed) Error() string {
	return "malformed " + err.stage
}
