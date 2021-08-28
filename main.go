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

// robot is an advanced Markov chain bot that operates on Twitch IRC.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/commands"
	"github.com/zephyrtronium/robot/dash"
	"github.com/zephyrtronium/robot/irc"
)

const copying = `
robot  Copyright (C) 2020  Branden J Brown
This program comes with ABSOLUTELY NO WARRANTY; for details type 'warranty'.
This is free software, and you are welcome to redistribute it
under certain conditions; see the GNU General Public License, Version 3,
for details.

`

func main() {
	var source, remote, token, dsh string
	var secure bool
	var checkp time.Duration
	var botlvl, echo string
	flag.StringVar(&source, "source", "", "SQL database source (required)")
	flag.StringVar(&remote, "remote", "irc.chat.twitch.tv:6697", "remote address, IRC protocol")
	flag.StringVar(&token, "token", "", "OAuth token")
	flag.StringVar(&dsh, "dash", "", "dashboard address (no dash if not given)")
	flag.BoolVar(&secure, "secure", true, "use TLS")
	flag.DurationVar(&checkp, "period", time.Minute, "period between checking broadcaster online statuses")
	flag.StringVar(&botlvl, "level", "", `bot level, "" or "known" or "verified"`)
	flag.StringVar(&echo, "echo", "", "directory to echo generated messages (no echoing if not given)")
	flag.Parse()

	// Print GPLv3 information.
	os.Stderr.WriteString(copying)

	ctx, cancel := context.WithCancel(context.Background())
	br, err := brain.Open(ctx, source)
	if err != nil {
		log.Fatalln(err)
	}
	br.SetEchoDir(echo)
	switch botlvl {
	case "": // do nothing
	case "known":
		br.SetFallbackWait(rate.NewLimiter(10, 1), rate.NewLimiter(rate.Every(time.Minute), 200))
	case "verified":
		br.SetFallbackWait(rate.NewLimiter(20, 1), rate.NewLimiter(rate.Every(time.Minute), 1200))
	default:
		log.Fatalf(`unknown bot level %q, must be "" or "known" or "verified"`, botlvl)
	}
	cfg := connectConfig{
		addr:    remote,
		nick:    strings.ToLower(br.Name()),
		pass:    "oauth:" + token,
		timeout: 300 * time.Second,
		retries: []time.Duration{5 * time.Second, 15 * time.Second, time.Minute, time.Minute, 3 * time.Minute},
	}
	if secure {
		cfg.dialer = &tls.Dialer{}
	} else {
		cfg.dialer = &net.Dialer{}
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)
	go func() {
		<-sig
		cancel()
		signal.Stop(sig)
	}()
	go onlineLoop(ctx, br, token, checkp, log.New(os.Stderr, "(online)", log.Ltime))
	send := make(chan irc.Message)
	recv := make(chan irc.Message, 1)
	lg := log.New(os.Stderr, "(irc)", log.Ltime)
	go func() {
		connect(ctx, cfg, send, recv, lg)
		cancel()
	}()
	var wg sync.WaitGroup
	procs := runtime.GOMAXPROCS(0)
	wg.Add(procs + 2)
	go stdin(ctx, cancel, &wg, br, log.New(os.Stderr, "(stdin)", log.Ltime))
	lgp := log.New(os.Stderr, "(worker)", log.Ltime)
	for i := 0; i < procs; i++ {
		go loop(ctx, &wg, br, send, recv, lgp)
	}
	if dsh != "" {
		go func() {
			lg := log.New(os.Stderr, "(owner-dash)", log.Ltime)
			if err := dash.Owner(ctx, &wg, br, dsh, nil, lg); err != nil {
				lg.Fatalf("dashboard server error: %v", err)
			}
		}()
	}
	wg.Wait()
	br.Close()
}

func loop(ctx context.Context, wg *sync.WaitGroup, br *brain.Brain, send, recv chan irc.Message, lg *log.Logger) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-recv:
			if !ok {
				return
			}
			switch msg.Command {
			case "PRIVMSG":
				if err := privmsg(ctx, br, send, msg, lg); err != nil {
					lg.Println("error processing message:", err)
				}
			case "WHISPER":
				// TODO: this
			case "NOTICE":
				// nothing yet
			case "CLEARCHAT":
				// Delay forgetting messages to ensure that we finish learning
				// before removing what is learned.
				go func(msg irc.Message) {
					select {
					case <-ctx.Done():
						// It's more important to clear the message than to
						// shutdown quickly.
						br.ClearChat(context.Background(), msg.To(), msg.Nick)
						return
					case <-time.After(time.Second): // do nothing
					}
					if err := br.ClearChat(ctx, msg.To(), msg.Nick); err != nil {
						lg.Println("error clearing chat:", err)
					} else {
						lg.Println("cleared messages due to", msg.Text())
					}
				}(msg)
			case "CLEARMSG":
				id, ok := msg.Tag("target-msg-id")
				if !ok {
					lg.Println("??? CLEARMSG with no target-msg-id")
					break
				}
				go func(id string) {
					select {
					case <-ctx.Done():
						// It's more important to clear the message than to
						// shutdown quickly.
						br.ClearMsg(context.Background(), id)
						return
					case <-time.After(time.Second): // do nothing
					}
					if err := br.ClearMsg(ctx, id); err != nil {
						lg.Println("error clearing message:", err)
					} else {
						lg.Println("cleared message with id", id)
					}
				}(id)
			case "HOSTTARGET":
				// nothing yet
			case "USERSTATE":
				// Check our own badges and update the hard rate limit for this
				// channel.
				setWait(ctx, br, msg)
			case "376": // End MOTD
				ch := br.Channels()
				send <- irc.Message{Command: "JOIN", Params: []string{strings.Join(ch, ",")}}
			}
		}
	}
}

func privmsg(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, lg *log.Logger) error {
	var bb [4]string
	badges := msg.Badges(bb[:0])
	priv, err := br.Privilege(ctx, msg.To(), msg.Nick, badges)
	if err != nil {
		return err
	}
	if priv == "ignore" {
		return nil
	}
	if cmd, ok := commands.Parse(br.Name(), msg.Trailing); ok {
		nm := commands.Do(ctx, br, lg, send, msg, priv, cmd)
		if nm != "" {
			lg.Println("executed command", nm)
			return nil
		}
	}
	uid, _ := msg.Tag("user-id")
	if err := br.AddAffection(ctx, msg.To(), "", 1); err != nil {
		lg.Printf("couldn't add affection to everyone in %s: %v", msg.To(), err)
	}
	if br.ShouldTalk(ctx, msg, true) == nil {
		m := br.TalkIn(ctx, msg.To(), nil)
		if m != "" {
			eff := br.EffectIn(ctx, msg.To())
			if eff != "" {
				lg.Println("applying", eff, "to", m)
				m = commands.Effect(eff, m)
				if err := br.Said(ctx, msg.To(), m); err != nil {
					lg.Println("error marking message as said:", err)
				}
			}
			br.Wait(ctx, msg.To())
			if echo := br.EchoTo(msg.To()); echo != "" {
				go doEcho(ctx, lg, m, echo, msg.To())
			}
			send <- irc.Privmsg(msg.To(), m)
			if err := br.AddAffection(ctx, msg.To(), uid, 30); err != nil {
				lg.Println("couldn't add random chance affection:", err)
			}
		}
	}
	if err := br.Learn(ctx, msg); err != nil {
		lg.Println("error learning message:", err)
	}
	if err := br.CheckCopypasta(ctx, msg); err != nil {
		if !errors.Is(err, brain.NoCopypasta) {
			lg.Println(err)
			return nil
		}
	} else {
		// This message is copypasta.
		if err := br.ShouldTalk(ctx, msg, false); err != nil {
			lg.Println("won't copypasta:", err)
			return nil
		}
		eff := br.EffectIn(ctx, msg.To())
		cp := msg.Trailing
		if eff != "" {
			lg.Println("applying", eff, "to", msg.Trailing)
			cp = commands.Effect(eff, msg.Trailing)
		}
		send <- msg.Reply("%s", cp)
		if err := br.AddAffection(ctx, msg.To(), uid, 20); err != nil {
			lg.Println("couldn't add copypasta affection:", err)
		}
	}
	ok, err := br.DidSay(ctx, msg.To(), msg.Trailing)
	if err != nil {
		lg.Println(err)
	}
	if ok {
		if err := br.AddAffection(ctx, msg.To(), uid, 7); err != nil {
			lg.Println("couldn't add affection for copying me:", err)
		}
	}
	return nil
}

func stdin(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, br *brain.Brain, lg *log.Logger) {
	defer wg.Done()
	scan := bufio.NewScanner(os.Stdin)
	ch := make(chan string, 1)
	send := make(chan irc.Message, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-send:
				if !ok {
					return
				}
				lg.Println(msg)
			}
		}
	}()
	for {
		go func() {
			for {
				if !scan.Scan() {
					if scan.Err() == nil {
						// EOF, i.e. ^D at terminal. We're done here.
						cancel()
						return
					}
					lg.Println("scan error:", scan.Err())
					continue
				}
				break
			}
			ch <- scan.Text()
		}()
		select {
		case <-ctx.Done():
			return
		case cmd := <-ch:
			msg := irc.Message{
				Command:  "PRIVMSG",
				Params:   []string{"(terminal)"},
				Sender:   irc.Sender{Nick: "(terminal)"},
				Trailing: cmd,
			}
			nm := commands.Do(ctx, br, lg, send, msg, "owner", cmd)
			if nm != "" {
				lg.Println("executed command", nm)
			}
		}
	}
}

func onlineLoop(ctx context.Context, br *brain.Brain, token string, period time.Duration, lg *log.Logger) {
	clientID, err := fetchClientID(ctx, token)
	if err != nil {
		lg.Fatal("An error occurred while fetching client ID:", err)
	}
	lim := rate.NewLimiter(rate.Every(period), 1)
	var b strings.Builder
	for {
		if err := lim.Wait(ctx); err != nil {
			return
		}
		channels := br.Channels()
		on, err := online(ctx, token, clientID, channels)
		if err != nil {
			lg.Println(err)
			continue
		}
		b.Reset()
		for _, channel := range channels {
			b.WriteString(channel)
			if on[channel] {
				b.WriteString(" ONLINE  ")
			} else {
				b.WriteString(" offline  ")
				if err := br.ClearSince(ctx, channel, time.Now().Add(-period)); err != nil {
					lg.Printf("couldn't clear offline messages from %s: %v", channel, err)
				}
			}
			br.SetOnline(channel, on[channel])
		}
		lg.Println(b.String())
	}
}

func setWait(ctx context.Context, br *brain.Brain, msg irc.Message) {
	var bb [4]string
	badges := msg.Badges(bb[:0])
	for _, badge := range badges {
		switch badge {
		case "broadcaster", "moderator", "vip":
			br.SetWait(ctx, msg.To(), 100/30.0)
			return
		}
	}
	br.SetWait(ctx, msg.To(), 20/30.0)
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
