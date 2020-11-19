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
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/irc"
)

func help(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	cmd := findcmd(matches[1])
	if cmd == nil {
		selsend(ctx, br, send, br.Privmsg(ctx, msg.To(), fmt.Sprintf(`@%s couldn't find a command named %q`, msg.DisplayName(), matches[1])))
		return
	}
	selsend(ctx, br, send, br.Privmsg(ctx, msg.To(), cmd.help))
}

func invocation(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	cmd := findcmd(matches[1])
	if cmd == nil {
		selsend(ctx, br, send, br.Privmsg(ctx, msg.To(), fmt.Sprintf(`@%s couldn't find a command named %q`, msg.DisplayName(), matches[1])))
		return
	}
	selsend(ctx, br, send, msg.Reply("%s", cmd.re.String()))
}

func list(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	var r []string
	for _, cmd := range all {
		if cmd.enabled() && (cmd.admin || cmd.regular) {
			r = append(r, cmd.name)
		}
	}
	selsend(ctx, br, send, msg.Reply("%s", strings.Join(r, " ")))
}

func forget(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	n, err := br.ClearPattern(ctx, msg.To(), matches[1])
	if err != nil {
		selsend(ctx, br, send, msg.Reply(`@%s an error occurred while trying to forget: %v`, msg.DisplayName(), err))
	}
	selsend(ctx, br, send, br.Privmsg(ctx, msg.To(), fmt.Sprintf(`@%s deleted %d messages!`, msg.DisplayName(), n)))
}

func silence(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
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
				selsend(ctx, br, send, msg.Reply(`@%s sorry? (%v)`, msg.DisplayName(), err))
				return
			}
			until = msg.Time.Add(time.Duration(n) * time.Hour)
			break
		}
		if m := silenceNMin.FindStringSubmatch(matches[1]); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				selsend(ctx, br, send, msg.Reply(`@%s sorry? (%v)`, msg.DisplayName(), err))
				return
			}
			until = msg.Time.Add(time.Duration(n) * time.Minute)
			break
		}
		dur, err := time.ParseDuration(matches[1])
		if err != nil {
			selsend(ctx, br, send, msg.Reply(`@%s sorry? (%v)`, msg.DisplayName(), err))
			return
		}
		until = msg.Time.Add(dur)
	}
	if until.After(msg.Time.Add(12*time.Hour + time.Second)) {
		err := br.Silence(ctx, msg.To(), msg.Time.Add(12*time.Hour))
		if err != nil {
			selsend(ctx, br, send, msg.Reply(`@%s error setting silence: %v`, msg.DisplayName(), err))
			return
		}
		selsend(ctx, br, send, msg.Reply(`@%s silent for 12h. If you really need longer, contact the bot owner.`, msg.DisplayName()))
		return
	}
	if err := br.Silence(ctx, msg.To(), until); err != nil {
		selsend(ctx, br, send, msg.Reply(`@%s error setting silence: %v`, msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, send, msg.Reply(`@%s I won't randomly talk or learn for %v`, msg.DisplayName(), until.Sub(msg.Time)))
}

var (
	silenceAnHr = regexp.MustCompile(`(?i)^an\s+h(?:ou)?r`)
	silenceNHrs = regexp.MustCompile(`(?i)^(\d+)\s+h(?:ou)?rs?`)
	silenceAMin = regexp.MustCompile(`(?i)^a\s+min`)
	silenceNMin = regexp.MustCompile(`(?i)^(\d+)\s+min`)
)

func unsilence(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	if err := br.Silence(ctx, msg.To(), time.Time{}); err != nil {
		selsend(ctx, br, send, msg.Reply(`@%s error removing silence: %v`, msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, send, msg.Reply(`@%s thanks for letting me talk again!`, msg.DisplayName()))
}

func tooActive(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	p, err := br.Activity(ctx, msg.To(), func(x float64) float64 { return x / 2 })
	if err != nil {
		selsend(ctx, br, send, msg.Reply(`@%s error setting activity: %v`, msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, send, msg.Reply(`@%s response rate set to %g%%`, msg.DisplayName(), p*100))
}

func setProb(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	p, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		selsend(ctx, br, send, msg.Reply(`@%s didn't understand %s: %v`, msg.DisplayName(), matches[1], err))
		return
	}
	p *= 0.01 // always use percentages
	if _, err := br.Activity(ctx, msg.To(), func(float64) float64 { return p }); err != nil {
		selsend(ctx, br, send, msg.Reply(`@%s error setting activity: %v`, msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, send, msg.Reply(`@%s response rate set to %g%%!`, msg.DisplayName(), p*100))
}

func multigen(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	n, err := strconv.Atoi(matches[1])
	if err != nil {
		selsend(ctx, br, send, msg.Reply(`@%s didn't understand %s: %v`, msg.DisplayName(), matches[1], err))
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
		selsend(ctx, br, send, msg.Reply("%s", m))
		time.Sleep(1*time.Second + 15*time.Millisecond)
	}
}

func raid(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	multigen(ctx, br, lg, send, msg, []string{"", "5"})
}

func givePrivacyAdmin(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	who := strings.ToLower(msg.DisplayName())
	if !strings.HasPrefix(msg.To(), "#") {
		selsend(ctx, br, send, msg.Reply(`@%s sorry, I can't modify your privileges in whispers. Contact the bot owner for help.`, msg.DisplayName()))
		return
	}
	if err := br.SetPriv(ctx, who, msg.To(), "bot"); err != nil {
		selsend(ctx, br, send, msg.Reply(`@%s an error occurred: %v. Contact the bot owner for help.`, msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, send, msg.Reply(`@%s got it, I won't record any of your messages in %s.`, msg.DisplayName(), msg.To()))
}

func removePrivacyAdmin(ctx context.Context, br *brain.Brain, lg *log.Logger, send chan<- irc.Message, msg irc.Message, matches []string) {
	who := strings.ToLower(msg.DisplayName())
	if !strings.HasPrefix(msg.To(), "#") {
		selsend(ctx, br, send, msg.Reply(`@%s sorry, I can't modify your privileges in whispers. Contact the bot owner for help.`, msg.DisplayName()))
		return
	}
	if err := br.SetPriv(ctx, who, msg.To(), "admin"); err != nil {
		selsend(ctx, br, send, msg.Reply(`@%s an error occurred: %v. Contact the bot owner for help.`, msg.DisplayName(), err))
		return
	}
	selsend(ctx, br, send, msg.Reply(`@%s got it, I'll learn from your messages in %s again.`, msg.DisplayName(), msg.To()))
}
