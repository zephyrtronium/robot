package commands

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/irc"
)

func enable(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	cmd := findcmd(matches[2])
	if cmd == nil {
		selsend(ctx, send, msg.Reply(fmt.Sprintf("@%s didn't find a command named %q", msg.Nick, matches[2])))
		return
	}
	if strings.EqualFold(matches[1], "disable") {
		atomic.StoreInt32(&cmd.disable, 1)
		selsend(ctx, send, msg.Reply(fmt.Sprintf("@%s disabled!", msg.Nick)))
	} else {
		atomic.StoreInt32(&cmd.disable, 0)
		selsend(ctx, send, msg.Reply(fmt.Sprintf("@%s enabled!", msg.Nick)))
	}
}

func resync(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	err := br.UpdateAll(ctx)
	if err != nil {
		selsend(ctx, send, msg.Reply(fmt.Sprintf("@%s error from UpdateAll: %v", msg.Nick, err)))
		return
	}
	selsend(ctx, send, msg.Reply(fmt.Sprintf("@%s updated!", msg.Nick)))
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
		selsend(ctx, send, msg.Reply(fmt.Sprintf("@%s error from Join: %v", msg.Nick, err)))
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
		selsend(ctx, send, msg.Reply(fmt.Sprintf(`@%s error from SetPriv: %v`, msg.Nick, err)))
		return
	}
	selsend(ctx, send, msg.Reply(fmt.Sprintf(`@%s set privs for %s!`, msg.Nick, matches[1])))
	if priv != "ignore" || where == "" {
		return
	}
	if err := br.ClearChat(ctx, where, who); err != nil {
		selsend(ctx, send, msg.Reply(fmt.Sprintf(`@%s couldn't delete their messages: %v`, msg.Nick, err)))
	}
}

func exec(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	res, err := br.Exec(ctx, matches[1])
	if err != nil {
		selsend(ctx, send, msg.Reply(fmt.Sprintf(`@%s error from Exec: %v`, msg.Nick, err)))
		return
	}
	n, err := res.RowsAffected()
	if err != nil {
		selsend(ctx, send, msg.Reply(fmt.Sprintf(`@%s error from sql.Result.RowsAffected:`, err)))
		// Don't return. Worst case, there's an extra @ with "0 rows modified."
	}
	selsend(ctx, send, msg.Reply(fmt.Sprintf(`@%s your query modified %d rows`, msg.Nick, n)))
}

func quit(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	selsend(ctx, send, irc.Message{Command: "QUIT", Trailing: "goodbye"})
}
