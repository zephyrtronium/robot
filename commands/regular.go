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

package commands

import (
	"context"
	"io/ioutil"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/irc"
)

func talk(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	if err := br.ShouldTalk(ctx, msg, false); err != nil {
		lg.Println("won't talk:", err)
		return
	}
	with := strings.TrimSpace(matches[1])
	chk := func() bool {
		if len(with) <= 1 {
			return false
		}
		if with[0] != '/' && with[0] != '.' {
			return false
		}
		if r, _ := utf8.DecodeRuneInString(with[1:]); !unicode.IsLetter(r) {
			return false
		}
		selsend(ctx, br, send, br.Privmsg(ctx, msg.To(), "no"))
		return true
	}
	if chk() {
		return
	}
	toks := brain.Tokens(with)
	m := br.TalkIn(ctx, msg.To(), toks)
	if m == "" {
		return
	}
	if echo := br.EchoTo(msg.To()); echo != "" {
		go doEcho(ctx, lg, m, echo, msg.To())
	}
	selsend(ctx, br, send, msg.Reply("%s", m))
}

func talkCatchall(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	if err := br.ShouldTalk(ctx, msg, false); err != nil {
		lg.Println("won't talk:", err)
		return
	}
	m := br.TalkIn(ctx, msg.To(), nil)
	if echo := br.EchoTo(msg.To()); echo != "" {
		go doEcho(ctx, lg, m, echo, msg.To())
	}
	selsend(ctx, br, send, msg.Reply("%s", m))
}

func uwu(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	if err := br.ShouldTalk(ctx, msg, false); err != nil {
		lg.Println("won't talk:", err)
		return
	}
	m := br.TalkIn(ctx, msg.To(), nil)
	if m == "" {
		return
	}
	m = uwuRep.Replace(m)
	if err := br.Said(ctx, msg.To(), m); err != nil {
		lg.Println("error marking message as said:", err)
	}
	if echo := br.EchoTo(msg.To()); echo != "" {
		go doEcho(ctx, lg, m, echo, msg.To())
	}
	selsend(ctx, br, send, msg.Reply("%s", m))
}

var uwuRep = strings.NewReplacer(
	"r", "w", "R", "W",
	"l", "w", "L", "W",
	"na", "nya", "Na", "Nya", "NA", "NYA",
	"ni", "nyi", "Ni", "Nyi", "NI", "NYI",
	"nu", "nyu", "Nu", "Nyu", "NU", "NYU",
	"ne", "nye", "Ne", "Nye", "NE", "NYE",
	"no", "nyo", "No", "Nyo", "NO", "NYO",
)

// AAAAA AAAAAAAAA A AAAAAAA AAA AAAAAAAA AAA AAA AAAAAAA AAAA AA.
func AAAAA(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	if err := br.ShouldTalk(ctx, msg, false); err != nil {
		lg.Println("won't talk:", err)
		return
	}
	tag, ok := br.SendTag(msg.To())
	if !ok {
		return
	}
	m := br.Talk(ctx, tag, nil, 40)
	if m == "" {
		return
	}
	m = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return 'A'
		}
		return r
	}, m)
	// m = aaaaaRe.ReplaceAllString(m, "${1}H!")
	if err := br.Said(ctx, msg.To(), m); err != nil {
		lg.Println("error marking message as said:", err)
	}
	if echo := br.EchoTo(msg.To()); echo != "" {
		go doEcho(ctx, lg, m, echo, msg.To())
	}
	selsend(ctx, br, send, msg.Reply("%s", m))
}

// var aaaaaRe = regexp.MustCompile(`\b(A+)AA\b`)

// doEcho writes a message as a file to echo.
func doEcho(ctx context.Context, lg *log.Logger, msg, echo, channel string) {
	f, err := ioutil.TempFile(echo, channel)
	if err != nil {
		lg.Println("couldn't open echo file:", err)
		return
	}
	if _, err := f.WriteString(msg); err != nil {
		lg.Println("couldn't write message to echo file:", err)
		return
	}
	if err := f.Close(); err != nil {
		lg.Println("error closing echo file:", err)
		return
	}
}

func source(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	// We could try to extract the package path from a function name or
	// something, or we can just do this.
	selsend(ctx, br, send, msg.Reply(`@%s My source code is at https://github.com/zephyrtronium/robot – `+
		`I'm written in Go leveraging SQLite3, `+
		`and I'm free, open-source software licensed `+
		`under the GNU General Public License, Version 3.`, msg.DisplayName()))
}

func givePrivacy(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	who := strings.ToLower(msg.Nick)
	if err := br.SetPriv(ctx, who, "", "privacy"); err != nil {
		selsend(ctx, br, send, msg.Reply(`@%s an error occurred: %v. Contact the bot owner for help.`, msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, send, msg.Reply(`@%s got it, I won't record any of your messages.`, msg.DisplayName()))
}

func removePrivacy(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	who := strings.ToLower(msg.Nick)
	if err := br.SetPriv(ctx, who, "", ""); err != nil {
		selsend(ctx, br, send, msg.Reply(`@%s an error occurred: %v. Contact the bot owner for help.`, msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, send, msg.Reply(`@%s got it, I'll learn from your messages again.`, msg.DisplayName()))
}

func describePrivacy(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	selsend(ctx, br, send, msg.Reply(`@%s see here for info about what information I collect, and how to opt out of all collection: https://github.com/zephyrtronium/robot#what-information-does-robot-store`, msg.DisplayName()))
}
