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
	"os"
	"strings"
	"sync/atomic"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/irc"
)

func enable(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	cmd := findcmd(matches[2])
	if cmd == nil {
		selsend(ctx, send, msg.Reply("@%s didn't find a command named %q", msg.Nick, matches[2]))
		return
	}
	if strings.EqualFold(matches[1], "disable") {
		atomic.StoreInt32(&cmd.disable, 1)
		selsend(ctx, send, msg.Reply("@%s disabled!", msg.Nick))
	} else {
		atomic.StoreInt32(&cmd.disable, 0)
		selsend(ctx, send, msg.Reply("@%s enabled!", msg.Nick))
	}
}

func resync(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	err := br.UpdateAll(ctx)
	if err != nil {
		selsend(ctx, send, msg.Reply("@%s error from UpdateAll: %v", msg.Nick, err))
		return
	}
	selsend(ctx, send, msg.Reply("@%s updated!", msg.Nick))
}

func raw(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	m := irc.Message{
		Command:  matches[1],
		Params:   strings.Fields(matches[2]),
		Trailing: matches[3],
	}
	selsend(ctx, send, m)
}

func join(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	if err := br.Join(ctx, matches[1], matches[2], matches[3]); err != nil {
		selsend(ctx, send, msg.Reply("@%s error from Join: %v", msg.Nick, err))
		return
	}
	selsend(ctx, send, irc.Message{Command: "JOIN", Params: []string{matches[1]}})
}

func privs(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	who := strings.ToLower(matches[1])
	priv := matches[2]
	if priv == "regular" {
		priv = ""
	}
	where := msg.To()
	switch matches[3] {
	case "": // do nothing
	case "everywhere":
		where = ""
	default:
		where = matches[3]
	}
	if err := br.SetPriv(ctx, who, where, priv); err != nil {
		selsend(ctx, send, msg.Reply(`@%s error from SetPriv: %v`, msg.Nick, err))
		return
	}
	selsend(ctx, send, msg.Reply(`@%s set privs for %s!`, msg.Nick, matches[1]))
	if priv != "ignore" || where == "" {
		return
	}
	if err := br.ClearChat(ctx, where, who); err != nil {
		selsend(ctx, send, msg.Reply(`@%s couldn't delete their messages: %v`, msg.Nick, err))
	}
}

func exec(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	res, err := br.Exec(ctx, matches[1])
	if err != nil {
		selsend(ctx, send, msg.Reply(`@%s error from Exec: %v`, msg.Nick, err))
		return
	}
	n, err := res.RowsAffected()
	if err != nil {
		selsend(ctx, send, msg.Reply(`@%s error from sql.Result.RowsAffected:`, err))
		// Don't return. Worst case, there's an extra @ with "0 rows modified."
	}
	if err := br.UpdateAll(ctx); err != nil {
		selsend(ctx, send, msg.Reply(`@%s your query modified %d rows, but couldn't resync: %v`, msg.Nick, n, err))
		return
	}
	selsend(ctx, send, msg.Reply(`@%s your query modified %d rows`, msg.Nick, n))
}

func quit(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	selsend(ctx, send, irc.Message{Command: "QUIT", Trailing: "goodbye"})
}

func warranty(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	os.Stderr.WriteString(warrantyText)
}

const warrantyText = `

  THERE IS NO WARRANTY FOR THE PROGRAM, TO THE EXTENT PERMITTED BY
APPLICABLE LAW.  EXCEPT WHEN OTHERWISE STATED IN WRITING THE COPYRIGHT
HOLDERS AND/OR OTHER PARTIES PROVIDE THE PROGRAM "AS IS" WITHOUT WARRANTY
OF ANY KIND, EITHER EXPRESSED OR IMPLIED, INCLUDING, BUT NOT LIMITED TO,
THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
PURPOSE.  THE ENTIRE RISK AS TO THE QUALITY AND PERFORMANCE OF THE PROGRAM
IS WITH YOU.  SHOULD THE PROGRAM PROVE DEFECTIVE, YOU ASSUME THE COST OF
ALL NECESSARY SERVICING, REPAIR OR CORRECTION.

`

func reconnect(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	selsend(ctx, send, irc.Message{Command: "RECONNECT"})
}

func listOwner(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	var r []string
	for _, cmd := range all {
		if cmd.enabled() {
			r = append(r, cmd.name)
		} else {
			r = append(r, cmd.name+"*")
		}
	}
	selsend(ctx, send, msg.Reply(strings.Join(r, " ")))
}
