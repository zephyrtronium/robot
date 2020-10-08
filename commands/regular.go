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
	"strings"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/irc"
)

func talk(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	if !br.ShouldTalk(ctx, msg, false) {
		return
	}
	with := strings.TrimSpace(matches[1])
	toks := brain.Tokens(with)
	m := br.TalkIn(ctx, msg.To(), toks)
	if m == "" {
		return
	}
	selsend(ctx, send, msg.Reply("%s", m))
}

func talkCatchall(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	if !br.ShouldTalk(ctx, msg, false) {
		return
	}
	selsend(ctx, send, msg.Reply("%s", br.TalkIn(ctx, msg.To(), nil)))
}

func uwu(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	if !br.ShouldTalk(ctx, msg, false) {
		return
	}
	m := br.TalkIn(ctx, msg.To(), nil)
	if m == "" {
		return
	}
	selsend(ctx, send, msg.Reply("%s", uwuRep.Replace(m)))
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

func source(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	// We could try to extract the package path from a function name or
	// something, or we can just do this.
	selsend(ctx, send, msg.Reply(`@%s My source code is at https://github.com/zephyrtronium/robot – `+
		`I'm written in Go leveraging SQLite3, `+
		`and I'm free, open-source software licensed `+
		`under the GNU General Public License, Version 3.`, msg.Nick))
}
