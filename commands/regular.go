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
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/zephyrtronium/robot/brain"
)

func talk(ctx context.Context, call *call) {
	br := call.br
	if err := br.ShouldTalk(ctx, call.msg, false); err != nil {
		call.lg.Println("won't talk:", err)
		return
	}
	with := strings.TrimSpace(call.matches[1])
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
		selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), "no"))
		return true
	}
	if chk() {
		return
	}
	toks := brain.Tokens(with)
	m := br.TalkIn(ctx, call.msg.To(), toks)
	if m == "" {
		return
	}
	if echo := br.EchoTo(call.msg.To()); echo != "" {
		go doEcho(ctx, call.lg, m, echo, call.msg.To())
	}
	selsend(ctx, br, call.send, call.msg.Reply("%s", m))
	if len(toks) == 0 {
		uid, _ := call.msg.Tag("user-id")
		if err := br.AddAffection(ctx, call.msg.To(), uid, 2); err != nil {
			call.lg.Println("couldn't add affection:", err)
		}
	}
}

func talkCatchall(ctx context.Context, call *call) {
	br := call.br
	if err := br.ShouldTalk(ctx, call.msg, false); err != nil {
		call.lg.Println("won't talk:", err)
		return
	}
	m := br.TalkIn(ctx, call.msg.To(), nil)
	if echo := br.EchoTo(call.msg.To()); echo != "" {
		go doEcho(ctx, call.lg, m, echo, call.msg.To())
	}
	selsend(ctx, br, call.send, call.msg.Reply("%s", m))
	uid, _ := call.msg.Tag("user-id")
	if err := br.AddAffection(ctx, call.msg.To(), uid, 2); err != nil {
		call.lg.Println("couldn't add affection:", err)
	}
}

func uwu(ctx context.Context, call *call) {
	br := call.br
	if err := br.ShouldTalk(ctx, call.msg, false); err != nil {
		call.lg.Println("won't talk:", err)
		return
	}
	m := br.TalkIn(ctx, call.msg.To(), nil)
	if m == "" {
		return
	}
	m = uwuRep.Replace(m)
	if err := br.Said(ctx, call.msg.To(), m); err != nil {
		call.lg.Println("error marking message as said:", err)
	}
	if echo := br.EchoTo(call.msg.To()); echo != "" {
		go doEcho(ctx, call.lg, m, echo, call.msg.To())
	}
	selsend(ctx, br, call.send, call.msg.Reply("%s", m))
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
func AAAAA(ctx context.Context, call *call) {
	br := call.br
	if err := br.ShouldTalk(ctx, call.msg, false); err != nil {
		call.lg.Println("won't talk:", err)
		return
	}
	tag, ok := br.SendTag(call.msg.To())
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
	if err := br.Said(ctx, call.msg.To(), m); err != nil {
		call.lg.Println("error marking message as said:", err)
	}
	if echo := br.EchoTo(call.msg.To()); echo != "" {
		go doEcho(ctx, call.lg, m, echo, call.msg.To())
	}
	selsend(ctx, br, call.send, call.msg.Reply("%s", m))
}

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

func source(ctx context.Context, call *call) {
	br := call.br
	// We could try to extract the package path from a function name or
	// something, or we can just do this.
	selsend(ctx, br, call.send, call.msg.Reply(`@%s My source code is at https://github.com/zephyrtronium/robot – `+
		`I'm written in Go leveraging SQLite3, `+
		`and I'm free, open-source software licensed `+
		`under the GNU General Public License, Version 3.`, call.msg.DisplayName()))
}

func givePrivacy(ctx context.Context, call *call) {
	br := call.br
	who := strings.ToLower(call.msg.Nick)
	if err := br.SetPriv(ctx, who, "", "privacy"); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s an error occurred: %v. Contact the bot owner for help.`, call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply(`@%s got it, I won't record any of your messages.`, call.msg.DisplayName()))
}

func removePrivacy(ctx context.Context, call *call) {
	br := call.br
	who := strings.ToLower(call.msg.Nick)
	if err := br.SetPriv(ctx, who, "", ""); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s an error occurred: %v. Contact the bot owner for help.`, call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply(`@%s got it, I'll learn from your messages again.`, call.msg.DisplayName()))
}

func describePrivacy(ctx context.Context, call *call) {
	br := call.br
	selsend(ctx, br, call.send, call.msg.Reply(`@%s see here for info about what information I collect, and how to opt out of all collection: https://github.com/zephyrtronium/robot#what-information-does-robot-store`, call.msg.DisplayName()))
}

func roar(ctx context.Context, call *call) {
	br := call.br
	if err := br.ShouldTalk(ctx, call.msg, false); err != nil {
		call.lg.Println("won't talk:", err)
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply("rawr ;3"))
}

func marry(ctx context.Context, call *call) {
	br := call.br
	if err := br.ShouldTalk(ctx, call.msg, false); err != nil {
		call.lg.Println("won't talk:", err)
		return
	}
	priv, err := br.Privilege(ctx, call.msg.To(), call.msg.Nick, call.msg.Badges(nil))
	if err != nil {
		call.lg.Printf("error getting priviliges for %s in %s: %v", call.msg.Nick, call.msg.To(), err)
		return
	}
	if priv == "privacy" || priv == "bot" {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s Sorry, this feature requires recording some information about you, which your privacy level prohibits me from doing.`, call.msg.DisplayName()))
		return
	}
	uid, ok := call.msg.Tag("user-id")
	if !ok {
		call.lg.Println("no user-id:", call.msg)
		return
	}
	new, err := br.TrackAffection(ctx, call.msg.To(), uid)
	if err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s there was a problem adding you to my suitors: %v`, call.msg.DisplayName(), err))
		return
	}
	if new {
		selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s Interesting. I suppose I could add you to my list. Good luck, little one. You're going to need it.`, call.msg.DisplayName())))
		return
	}
	score, err := br.Affection(ctx, call.msg.To(), uid)
	if err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s there was a problem checking how much I like you: %v`, err))
		return
	}
	if score < 50 {
		selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s no`, call.msg.DisplayName())))
		return
	}
	// Marriage is not atomic, but I'm not terribly concerned about it.
	cur, since, beat, err := br.Marriage(ctx, call.msg.To())
	if cur == "" {
		if err := br.Marry(ctx, call.msg.To(), uid, call.msg.Time); err != nil {
			selsend(ctx, br, call.send, call.msg.Reply(`@%s I tried, but couldn't: %v`, call.msg.DisplayName(), err))
			return
		}
		selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s sure why not`, call.msg.DisplayName())))
		return
	}
	if uid == cur {
		switch call.msg.Time.Nanosecond() / 1e7 % 5 {
		case 0, 1, 3:
			selsend(ctx, br, call.send, call.msg.Reply(`@%s How could you forget we're already together? I hate you! Unsubbed, unfollowed, unloved!`, call.msg.DisplayName()))
			if err := br.Marry(ctx, call.msg.To(), "", call.msg.Time); err != nil {
				call.lg.Printf("error divorcing in %s: %v", call.msg.To(), err)
			}
			if err := br.AddAffection(ctx, call.msg.To(), cur, -beat/4); err != nil {
				call.lg.Printf("error dropping score for %s uid=%s in %s: %v", call.msg.DisplayName(), cur, call.msg.To(), err)
			}
		default:
			selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s We're already together, silly! You're so funny and cute haha.`, call.msg.DisplayName())))
			if err := br.AddAffection(ctx, call.msg.To(), cur, 1); err != nil {
				call.lg.Println(err)
			}
		}
		return
	}
	if call.msg.Time.Sub(since) < 1*time.Hour {
		selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s My heart yet belongs to another...`, call.msg.DisplayName())))
		return
	}
	if score < beat-beat/8 {
		selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s I'm touched, but I must decline. I'm in love with someone else.`, call.msg.DisplayName())))
		if err := br.AddAffection(ctx, call.msg.To(), cur, 1); err != nil {
			call.lg.Println(err)
		}
		return
	}
	if err := br.Marry(ctx, call.msg.To(), uid, call.msg.Time); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s I tried, but couldn't: %v`, call.msg.DisplayName(), err))
		return
	}
	if call.matches[1] == "partner" {
		selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s Yes! I'll be your partner!`, call.msg.DisplayName())))
		return
	}
	selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s Yes! I'll marry you!`, call.msg.DisplayName())))
}

func affection(ctx context.Context, call *call) {
	br := call.br
	if err := br.ShouldTalk(ctx, call.msg, false); err != nil {
		call.lg.Println("won't talk:", err)
		return
	}
	uid, _ := call.msg.Tag("user-id")
	score, err := br.Affection(ctx, call.msg.To(), uid)
	if err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s there was a problem checking how much I like you: %v`, call.msg.DisplayName(), err))
		return
	}
	if score <= 0 {
		selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s literally zero`, call.msg.DisplayName())))
		return
	}
	selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s about %f`, call.msg.DisplayName(), math.Log2(float64(score)))))
}
