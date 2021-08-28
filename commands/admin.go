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
	"regexp"
	"strconv"
	"strings"
	"time"
)

func help(ctx context.Context, call *call) {
	br := call.br
	cmd := findcmd(call.matches[1])
	if cmd == nil {
		selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s couldn't find a command named %q`, call.msg.DisplayName(), call.matches[1])))
		return
	}
	selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), cmd.help))
}

func invocation(ctx context.Context, call *call) {
	br := call.br
	cmd := findcmd(call.matches[1])
	if cmd == nil {
		selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s couldn't find a command named %q`, call.msg.DisplayName(), call.matches[1])))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply("%s", cmd.re.String()))
}

func list(ctx context.Context, call *call) {
	br := call.br
	var r []string
	for _, cmd := range all {
		if cmd.enabled() && (cmd.admin || cmd.regular) {
			r = append(r, cmd.name)
		}
	}
	selsend(ctx, br, call.send, call.msg.Reply("%s", strings.Join(r, " ")))
}

func forget(ctx context.Context, call *call) {
	br := call.br
	n, err := br.ClearPattern(ctx, call.msg.To(), call.matches[1])
	if err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s an error occurred while trying to forget: %v`, call.msg.DisplayName(), err))
	}
	selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), fmt.Sprintf(`@%s deleted %d messages!`, call.msg.DisplayName(), n)))
}

func silence(ctx context.Context, call *call) {
	br := call.br
	var until time.Time
	switch {
	case call.matches[1] == "" && call.matches[2] == "":
		until = call.msg.Time.Add(time.Hour)
	case call.matches[1] == "":
		// The only "until" option right now is "tomorrow".
		until = call.msg.Time.Add(12 * time.Hour)
	default:
		// We can do 1h2m3s, an hour, n hours, a minute, n minutes.
		if silenceAnHr.MatchString(call.matches[1]) {
			until = call.msg.Time.Add(time.Hour)
			break
		}
		if silenceAMin.MatchString(call.matches[1]) {
			until = call.msg.Time.Add(time.Minute)
			break
		}
		if m := silenceNHrs.FindStringSubmatch(call.matches[1]); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				selsend(ctx, br, call.send, call.msg.Reply(`@%s sorry? (%v)`, call.msg.DisplayName(), err))
				return
			}
			until = call.msg.Time.Add(time.Duration(n) * time.Hour)
			break
		}
		if m := silenceNMin.FindStringSubmatch(call.matches[1]); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				selsend(ctx, br, call.send, call.msg.Reply(`@%s sorry? (%v)`, call.msg.DisplayName(), err))
				return
			}
			until = call.msg.Time.Add(time.Duration(n) * time.Minute)
			break
		}
		dur, err := time.ParseDuration(call.matches[1])
		if err != nil {
			selsend(ctx, br, call.send, call.msg.Reply(`@%s sorry? (%v)`, call.msg.DisplayName(), err))
			return
		}
		until = call.msg.Time.Add(dur)
	}
	if until.After(call.msg.Time.Add(12*time.Hour + time.Second)) {
		err := br.Silence(ctx, call.msg.To(), call.msg.Time.Add(12*time.Hour))
		if err != nil {
			selsend(ctx, br, call.send, call.msg.Reply(`@%s error setting silence: %v`, call.msg.DisplayName(), err))
			return
		}
		selsend(ctx, br, call.send, call.msg.Reply(`@%s silent for 12h. If you really need longer, contact the bot owner.`, call.msg.DisplayName()))
		return
	}
	if err := br.Silence(ctx, call.msg.To(), until); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s error setting silence: %v`, call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply(`@%s I won't randomly talk or learn for %v`, call.msg.DisplayName(), until.Sub(call.msg.Time)))
}

var (
	silenceAnHr = regexp.MustCompile(`(?i)^an\s+h(?:ou)?r`)
	silenceNHrs = regexp.MustCompile(`(?i)^(\d+)\s+h(?:ou)?rs?`)
	silenceAMin = regexp.MustCompile(`(?i)^a\s+min`)
	silenceNMin = regexp.MustCompile(`(?i)^(\d+)\s+min`)
)

func unsilence(ctx context.Context, call *call) {
	br := call.br
	if err := br.Silence(ctx, call.msg.To(), time.Time{}); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s error removing silence: %v`, call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply(`@%s thanks for letting me talk again!`, call.msg.DisplayName()))
}

func tooActive(ctx context.Context, call *call) {
	br := call.br
	p, err := br.Activity(ctx, call.msg.To(), func(x float64) float64 { return x / 2 })
	if err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s error setting activity: %v`, call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply(`@%s response rate set to %g%%`, call.msg.DisplayName(), p*100))
}

func moreActive(ctx context.Context, call *call) {
	br := call.br
	p, err := br.Activity(ctx, call.msg.To(), func(x float64) float64 { return x + 0.01 })
	if err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s error setting activity: %v`, call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply(`@%s response rate set to %g%%`, call.msg.DisplayName(), p*100))
}

func setProb(ctx context.Context, call *call) {
	br := call.br
	p, err := strconv.ParseFloat(call.matches[1], 64)
	if err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s didn't understand %s: %v`, call.msg.DisplayName(), call.matches[1], err))
		return
	}
	p *= 0.01 // always use percentages
	if _, err := br.Activity(ctx, call.msg.To(), func(float64) float64 { return p }); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s error setting activity: %v`, call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply(`@%s response rate set to %g%%!`, call.msg.DisplayName(), p*100))
}

func multigen(ctx context.Context, call *call) {
	br := call.br
	n, err := strconv.Atoi(call.matches[1])
	if err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s didn't understand %s: %v`, call.msg.DisplayName(), call.matches[1], err))
		return
	}
	if n <= 0 {
		return
	}
	if n > 5 {
		n = 5
	}
	ch := make(chan string, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			m := br.TalkIn(ctx, call.msg.To(), nil)
			if i == 4 {
				m = uwuRep.Replace(m)
			}
			ch <- m
		}(i)
	}
	for i := 0; i < n; i++ {
		m := <-ch
		selsend(ctx, br, call.send, call.msg.Reply("%s", m))
	}
}

func raid(ctx context.Context, call *call) {
	call.matches = []string{"", "5"}
	multigen(ctx, call)
}

func givePrivacyAdmin(ctx context.Context, call *call) {
	br := call.br
	who := strings.ToLower(call.msg.DisplayName())
	if !strings.HasPrefix(call.msg.To(), "#") {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s sorry, I can't modify your privileges in whispers. Contact the bot owner for help.`, call.msg.DisplayName()))
		return
	}
	if err := br.SetPriv(ctx, who, call.msg.To(), "bot"); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s an error occurred: %v. Contact the bot owner for help.`, call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply(`@%s got it, I won't record any of your messages in %s.`, call.msg.DisplayName(), call.msg.To()))
}

func removePrivacyAdmin(ctx context.Context, call *call) {
	br := call.br
	who := strings.ToLower(call.msg.DisplayName())
	if !strings.HasPrefix(call.msg.To(), "#") {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s sorry, I can't modify your privileges in whispers. Contact the bot owner for help.`, call.msg.DisplayName()))
		return
	}
	if err := br.SetPriv(ctx, who, call.msg.To(), "admin"); err != nil {
		selsend(ctx, br, call.send, call.msg.Reply(`@%s an error occurred: %v. Contact the bot owner for help.`, call.msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, call.send, call.msg.Reply(`@%s got it, I'll learn from your messages in %s again.`, call.msg.DisplayName(), call.msg.To()))
}

func describeMarriage(ctx context.Context, call *call) {
	br := call.br
	const s = `I am looking for a long series of short-term relationships and am holding a ranked competitive how-much-I-like-you tournament to decide my suitors! Politely ask me to marry you (or become your partner) and I'll start tracking your score. I like copypasta, memes, and long walks in the chat.`
	selsend(ctx, br, call.send, br.Privmsg(ctx, call.msg.To(), s))
}

func echoAdmin(ctx context.Context, call *call) {
	br := call.br
	selsend(ctx, br, call.send, call.msg.Reply("%s", call.matches[1]))
}
