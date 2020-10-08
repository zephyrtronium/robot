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

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/irc"
)

func help(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	cmd := findcmd(matches[1])
	if cmd == nil {
		selsend(ctx, send, br.Privmsg(ctx, msg.To(), fmt.Sprintf(`@%s couldn't find a command named %q`, msg.Nick, matches[1])))
		return
	}
	selsend(ctx, send, br.Privmsg(ctx, msg.To(), cmd.help))
}

func invocation(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	cmd := findcmd(matches[1])
	if cmd == nil {
		selsend(ctx, send, br.Privmsg(ctx, msg.To(), fmt.Sprintf(`@%s couldn't find a command named %q`, msg.Nick, matches[1])))
		return
	}
	selsend(ctx, send, msg.Reply(cmd.re.String()))
}

func list(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	var r []string
	for _, cmd := range all {
		if cmd.enabled() && (cmd.admin || cmd.regular) {
			r = append(r, cmd.name)
		}
	}
	selsend(ctx, send, msg.Reply(strings.Join(r, " ")))
}

func forget(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	n, err := br.ClearPattern(ctx, msg.To(), matches[1])
	if err != nil {
		selsend(ctx, send, msg.Reply(`@%s an error occurred while trying to forget: %v`, msg.Nick, err))
	}
	selsend(ctx, send, br.Privmsg(ctx, msg.To(), fmt.Sprintf(`@%s deleted %d messages!`, msg.Nick, n)))
}

func silence(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	var until time.Time
	switch {
	case matches[1] == "" && matches[2] == "":
		until = msg.Time.Add(time.Hour)
	case matches[1] == "":
		// The only "until" option right now is "tomorrow".
		until = msg.Time.Add(12 * time.Hour)
	default:
		// We can do 1h2m3s, an hour, n hours, a minute, n minutes.
		if silenceAnHr.MatchString(matches[1]) {
			until = msg.Time.Add(time.Hour)
			break
		}
		if silenceAMin.MatchString(matches[1]) {
			until = msg.Time.Add(time.Minute)
			break
		}
		if m := silenceNHrs.FindStringSubmatch(matches[1]); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				selsend(ctx, send, msg.Reply(`@%s sorry? (%v)`, msg.Nick, err))
				return
			}
			until = msg.Time.Add(time.Duration(n) * time.Hour)
			break
		}
		if m := silenceNMin.FindStringSubmatch(matches[1]); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				selsend(ctx, send, msg.Reply(`@%s sorry? (%v)`, msg.Nick, err))
				return
			}
			until = msg.Time.Add(time.Duration(n) * time.Minute)
			break
		}
		dur, err := time.ParseDuration(matches[1])
		if err != nil {
			selsend(ctx, send, msg.Reply(`@%s sorry? (%v)`, msg.Nick, err))
			return
		}
		until = msg.Time.Add(dur)
	}
	if until.After(msg.Time.Add(12*time.Hour + time.Second)) {
		err := br.Silence(ctx, msg.To(), msg.Time.Add(12*time.Hour))
		if err != nil {
			selsend(ctx, send, msg.Reply(`@%s error setting silence: %v`, msg.Nick, err))
			return
		}
		selsend(ctx, send, msg.Reply(`@%s silent for 12h. If you really need longer, contact the bot owner.`, msg.Nick))
		return
	}
	if err := br.Silence(ctx, msg.To(), until); err != nil {
		selsend(ctx, send, msg.Reply(`@%s error setting silence: %v`, msg.Nick, err))
		return
	}
	selsend(ctx, send, msg.Reply(`@%s I won't randomly talk or learn for %v`, msg.Nick, until.Sub(msg.Time)))
}

var (
	silenceAnHr = regexp.MustCompile(`(?i)^an\s+h(?:ou)?r`)
	silenceNHrs = regexp.MustCompile(`(?i)^(\d+)\s+h(?:ou)?rs?`)
	silenceAMin = regexp.MustCompile(`(?i)^a\s+min`)
	silenceNMin = regexp.MustCompile(`(?i)^(\d+)\s+min`)
)

func unsilence(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	if err := br.Silence(ctx, msg.To(), time.Time{}); err != nil {
		selsend(ctx, send, msg.Reply(`@%s error removing silence: %v`, msg.Nick, err))
		return
	}
	selsend(ctx, send, msg.Reply(`@%s thanks for letting me talk again!`, msg.Nick))
}

func tooActive(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	p, err := br.Activity(ctx, msg.To(), func(x float64) float64 { return x / 2 })
	if err != nil {
		selsend(ctx, send, msg.Reply(`@%s error setting activity: %v`, msg.Nick, err))
		return
	}
	selsend(ctx, send, msg.Reply(`@%s response rate set to %g%%`, msg.Nick, p*100))
}

func setProb(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	p, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		selsend(ctx, send, msg.Reply(`@%s didn't understand %s: %v`, msg.Nick, matches[1], err))
		return
	}
	p *= 0.01 // always use percentages
	if _, err := br.Activity(ctx, msg.To(), func(float64) float64 { return p }); err != nil {
		selsend(ctx, send, msg.Reply(`@%s error setting activity: %v`, msg.Nick, err))
		return
	}
	selsend(ctx, send, msg.Reply(`@%s response rate set to %g%%!`, msg.Nick, p*100))
}

func multigen(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	n, err := strconv.Atoi(matches[1])
	if err != nil {
		selsend(ctx, send, msg.Reply(`@%s didn't understand %s: %v`, msg.Nick, matches[1], err))
		return
	}
	if n <= 0 {
		return
	}
	if n > 5 {
		n = 5
	}
	for i := 0; i < n; i++ {
		m := br.TalkIn(ctx, msg.To(), nil)
		if i == 4 {
			m = uwuRep.Replace(m)
		}
		selsend(ctx, send, msg.Reply(m))
	}
}

func raid(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, matches []string) {
	multigen(ctx, br, send, msg, []string{"", "5"})
}
