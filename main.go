package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/commands"
	"github.com/zephyrtronium/robot/irc"
)

func main() {
	var source, config, remote, token string
	var secure bool
	flag.StringVar(&source, "source", "", "SQL database source (required)")
	flag.StringVar(&config, "config", "", "initial configuration as nick:order, e.g. robot:3")
	flag.StringVar(&remote, "remote", "irc.chat.twitch.tv:6697", "remote address, IRC protocol")
	flag.StringVar(&token, "token", "", "OAuth token")
	flag.BoolVar(&secure, "secure", true, "use TLS")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	var br *brain.Brain
	if config == "" {
		var err error
		br, err = brain.Open(ctx, source)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		c := strings.IndexByte(config, ':')
		if c <= 0 || c >= len(config)-1 {
			log.Fatalf("invalid config %q, must be nick:order, e.g. \"robot:3\"\n", config)
		}
		me := config[:c]
		order, err := strconv.Atoi(config[c+1:])
		if err != nil {
			log.Fatalln("error parsing order from config:", err)
		}
		br, err = brain.Configure(ctx, source, me, order)
		if err != nil {
			log.Fatalln(err)
		}
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
	send := make(chan irc.Message)
	recv := make(chan irc.Message, 1)
	lg := log.New(os.Stderr, "(irc)", log.Ltime)
	go connect(ctx, cfg, send, recv, lg)
	var wg sync.WaitGroup
	procs := runtime.GOMAXPROCS(0)
	wg.Add(procs + 1)
	go stdin(ctx, cancel, &wg, br, log.New(os.Stderr, "(stdin)", log.Ltime))
	lgp := log.New(os.Stderr, "(worker)", log.Ltime)
	for i := 0; i < procs; i++ {
		go loop(ctx, &wg, br, send, recv, lgp)
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
				if err := br.ClearChat(ctx, msg.To(), msg.Nick); err != nil {
					lg.Println("error clearing chat:", err)
				}
			case "CLEARMSG":
				id, ok := msg.Tag("target-message-id")
				if !ok {
					lg.Println("??? CLEARMSG with no target-message-id")
				}
				if err := br.ClearMsg(ctx, id); err != nil {
					lg.Println("error clearing message:", err)
				}
			case "HOSTTARGET":
				// nothing yet
			case "376": // End MOTD
				ch := br.Channels()
				send <- irc.Message{Command: "JOIN", Params: []string{strings.Join(ch, ",")}}
			}
		}
	}
}

func privmsg(ctx context.Context, br *brain.Brain, send chan<- irc.Message, msg irc.Message, lg *log.Logger) error {
	badges, _ := msg.Tag("badges")
	priv, err := br.Privilege(ctx, msg.To(), msg.Nick, badges)
	if err != nil {
		return err
	}
	if priv == "ignore" {
		return nil
	}
	if cmd, ok := commands.Parse(br.Name(), msg.Trailing); ok {
		nm := commands.Do(ctx, br, send, msg, priv, cmd)
		if nm != "" {
			lg.Println("executed command", nm)
			return nil
		}
	}
	if br.ShouldTalk(ctx, msg, true) {
		m := br.TalkIn(ctx, msg.To(), nil)
		if m != "" {
			send <- irc.Privmsg(msg.To(), m)
		}
	}
	if err := br.Learn(ctx, msg); err != nil {
		lg.Println("error learning message:", err)
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
			nm := commands.Do(ctx, br, send, msg, "owner", cmd)
			if nm != "" {
				lg.Println("executed command", nm)
			}
		}
	}
}
