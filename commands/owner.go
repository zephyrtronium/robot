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

	"github.com/zephyrtronium/robot/irc"
)

func enable(ctx context.Context, call *call) {
	br := call.br
	cmd := findcmd(call.matches[2])
	if cmd == nil {
		selsend(ctx, br, call.send, call.msg.Reply("@%s didn't find a command named %q", call.msg.DisplayName(), call.matches[2]))
		return
	}
	if strings.EqualFold(call.matches[1], "disable") {
		atomic.StoreInt32(&cmd.disable, 1)
		selsend(ctx, br, call.send, call.msg.Reply("@%s disabled!", call.msg.DisplayName()))
	} else {
		atomic.StoreInt32(&cmd.disable, 0)
		selsend(ctx, br, call.send, call.msg.Reply("@%s enabled!", call.msg.DisplayName()))
	}
}

func resync(ctx context.Context, call *call) {
	br := call.br
	err := br.UpdateAll(ctx)
	if err != nil {
		selsend(ctx, br, call.send, call.msg.Reply("@%s error from UpdateAll: %v", call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply("@%s updated!", call.msg.DisplayName()))
}

func raw(ctx context.Context, call *call) {
	br := call.br
	m := irc.Message{
		Tags:     call.matches[1],
		Command:  call.matches[2],
		Params:   strings.Fields(call.matches[3]),
		Trailing: call.matches[4],
	}
	selsend(ctx, br, call.send, m)
}

func join(ctx context.Context, call *call) {
	br := call.br
	if err := br.Join(ctx, call.matches[1], call.matches[2], call.matches[3]); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply("@%s error from Join: %v", call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, irc.Message{Command: "JOIN", Params: []string{call.matches[1]}})
}

func privs(ctx context.Context, call *call) {
	br := call.br
	who := strings.ToLower(call.matches[1])
	priv := call.matches[2]
	if priv == "regular" {
		priv = ""
	}
	where := call.msg.To()
	switch call.matches[3] {
	case "": // do nothing
	case "everywhere":
		where = ""
	default:
		where = call.matches[3]
	}
	if err := br.SetPriv(ctx, who, where, priv); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s error from SetPriv: %v`, call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply(`@%s set privs for %s!`, call.msg.DisplayName(), call.matches[1]))
	if (priv != "ignore" && priv != "privacy") || where == "" {
		return
	}
	if err := br.ClearChat(ctx, where, who); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s couldn't delete their messages: %v`, call.msg.DisplayName(), err))
	}
}

func exec(ctx context.Context, call *call) {
	br := call.br
	res, err := br.Exec(ctx, call.matches[1])
	if err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s error from Exec: %v`, call.msg.DisplayName(), err))
		return
	}
	n, err := res.RowsAffected()
	if err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s error from sql.Result.RowsAffected:`, err))
		// Don't return. Worst case, there's an extra @ with "0 rows modified."
	}
	if err := br.UpdateAll(ctx); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s your query modified %d rows, but couldn't resync: %v`, call.msg.DisplayName(), n, err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply(`@%s your query modified %d rows`, call.msg.DisplayName(), n))
}

func quit(ctx context.Context, call *call) {
	br := call.br
	selsend(ctx, br, call.send, irc.Message{Command: "QUIT", Trailing: "goodbye"})
}

func warranty(ctx context.Context, call *call) {
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

func reconnect(ctx context.Context, call *call) {
	br := call.br
	selsend(ctx, br, call.send, irc.Message{Command: "RECONNECT"})
}

func listOwner(ctx context.Context, call *call) {
	br := call.br
	var r []string
	for _, cmd := range all {
		if cmd.enabled() {
			r = append(r, cmd.name)
		} else {
			r = append(r, cmd.name+"*")
		}
	}
	selsend(ctx, br, call.send, call.msg.Reply("%s", strings.Join(r, " ")))
}

func debugChan(ctx context.Context, call *call) {
	br := call.br
	where := call.matches[2]
	if where == "" {
		where = call.msg.To()
	}
	var status, block, privs string
	var emotes, effects []string
	var ss, sb, sp, se, sf bool
	switch strings.ToLower(call.matches[1]) {
	case "", "channel":
		status, block, privs = br.Debug(where)
		if status == "" {
			selsend(ctx, br, call.send, call.msg.Reply("@%s no such channel %s", call.msg.DisplayName(), where))
			return
		}
		ss = true
		sb = true
		sp = true
	case "tag":
		emotes, effects = br.DebugTag(where)
		if len(emotes) == 0 && len(effects) == 0 {
			selsend(ctx, br, call.send, call.msg.Reply("@%s no such tag %s (or no emotes or effects)", call.msg.DisplayName(), where))
			return
		}
		se = true
		sf = true
	case "status":
		status, _, _ = br.Debug(where)
		if status == "" {
			selsend(ctx, br, call.send, call.msg.Reply("@%s no such channel %s", call.msg.DisplayName(), where))
			return
		}
		ss = true
	case "block":
		_, block, _ = br.Debug(where)
		if block == "" {
			selsend(ctx, br, call.send, call.msg.Reply("@%s no such channel %s (or block is empty string)", call.msg.DisplayName(), where))
			return
		}
		sb = true
	case "priv", "privs", "privilege", "privileges":
		_, _, privs = br.Debug(where)
		if privs == "" {
			selsend(ctx, br, call.send, call.msg.Reply("@%s no such channel %s (or no privs if on terminal)", call.msg.DisplayName(), where))
			return
		}
		sp = true
	case "emotes":
		emotes, _ = br.DebugTag(where)
		if len(emotes) == 0 {
			selsend(ctx, br, call.send, call.msg.Reply("@%s no such tag %s (or no emotes)", call.msg.DisplayName(), where))
			return
		}
		se = true
	case "effects":
		_, effects = br.DebugTag(where)
		if len(effects) == 0 {
			selsend(ctx, br, call.send, call.msg.Reply("@%s no such tag %s (or no effects)", call.msg.DisplayName(), where))
			return
		}
		sf = true
	default:
		selsend(ctx, br, call.send, call.msg.Reply("@%s unhandled op %q??? unreachable", call.msg.DisplayName(), call.matches[1]))
		return
	}
	if ss {
		selsend(ctx, br, call.send, call.msg.Reply("@%s status: %s", call.msg.DisplayName(), status))
	}
	if sb {
		selsend(ctx, br, call.send, call.msg.Reply("@%s block: %s", call.msg.DisplayName(), block))
	}
	if sp {
		selsend(ctx, br, call.send, call.msg.Reply("@%s privs: %s", call.msg.DisplayName(), privs))
	}
	if se {
		selsend(ctx, br, call.send, call.msg.Reply("@%s emotes: %s", call.msg.DisplayName(), emotes))
	}
	if sf {
		selsend(ctx, br, call.send, call.msg.Reply("@%s effects: %s", call.msg.DisplayName(), effects))
	}
}

func testChan(ctx context.Context, call *call) {
	br := call.br
	channel := call.matches[1]
	switch {
	case strings.EqualFold(call.matches[2], "online"):
		br.SetOnline(channel, true)
		status, _, _ := br.Debug(channel)
		selsend(ctx, br, call.send, call.msg.Reply(`@%s set %s online, status: %s`, call.msg.DisplayName(), channel, status))
	case strings.EqualFold(call.matches[2], "offline"):
		br.SetOnline(channel, false)
		status, _, _ := br.Debug(channel)
		selsend(ctx, br, call.send, call.msg.Reply(`@%s set %s offline, status: %s`, call.msg.DisplayName(), channel, status))
	default:
		selsend(ctx, br, call.send, call.msg.Reply(`@%s unrecognized op`, call.msg.DisplayName()))
	}
}

func roarOwner(ctx context.Context, call *call) {
	br := call.br
	if err := br.ShouldTalk(ctx, call.msg, false); err != nil {
		call.lg.Println("won't talk:", err)
		return
	}
	if echo := br.EchoTo(call.msg.To()); echo != "" {
		go doEcho(ctx, call.lg, "roooaaaaarrrrrrr", echo, call.msg.To())
	}
	selsend(ctx, br, call.send, call.msg.Reply("rawr ;3"))
}

func echoline(ctx context.Context, call *call) {
	br := call.br
	if echo := br.EchoTo(call.msg.To()); echo != "" {
		doEcho(ctx, call.lg, call.matches[1], echo, call.msg.To())
	}
	selsend(ctx, br, call.send, call.msg.Reply("%s", call.matches[1]))
}

func setTag(ctx context.Context, call *call) {
	br := call.br
	kind := call.matches[1]
	tag := call.matches[2]
	where := call.matches[3]
	if where == "" {
		where = call.msg.To()
	}
	switch {
	case strings.EqualFold(kind, "learn"):
		r, err := br.Exec(ctx, `UPDATE chans SET learn=? WHERE name=?`, tag, where)
		if err != nil {
			selsend(ctx, br, call.send, call.msg.Reply("@%s couldn't set learn tag in %s: %v", call.msg.DisplayName(), where, err))
			return
		}
		if n, _ := r.RowsAffected(); n == 0 {
			selsend(ctx, br, call.send, call.msg.Reply("@%s setting learn tag in %s didn't change any rows", call.msg.DisplayName(), where))
			return
		}
		selsend(ctx, br, call.send, call.msg.Reply("@%s learn tag in %s set to %q", call.msg.DisplayName(), where, tag))
	case strings.EqualFold(kind, "send"):
		r, err := br.Exec(ctx, `UPDATE chans SET send=? WHERE name=?`, tag, where)
		if err != nil {
			selsend(ctx, br, call.send, call.msg.Reply("@%s couldn't set send tag in %s: %v", call.msg.DisplayName(), where, err))
			return
		}
		if n, _ := r.RowsAffected(); n == 0 {
			selsend(ctx, br, call.send, call.msg.Reply("@%s setting send tag in %s didn't change any rows", call.msg.DisplayName(), where))
			return
		}
		selsend(ctx, br, call.send, call.msg.Reply("@%s send tag in %s set to %q", call.msg.DisplayName(), where, tag))
	}
	if err := br.Update(ctx, where); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s resync didn't work though`, call.msg.DisplayName()))
	}
}
