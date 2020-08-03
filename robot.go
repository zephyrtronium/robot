package main

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"github.com/zephyrtronium/crazy"
)

const (
	Server      = "irc.twitch.tv:6667"
	Nick        = "robotisbroken"
	User        = "robotisbroken"
	Real        = "robotisbroken"
	Channel     = "#brokencowleg,#robotisbroken"
	Listen      = ""
	Ignore      = "zbot,nightbot"
	RegexIgnore = `^!|://|\.(com|net|org|tv|edu)|^\001`
	Admins      = "brokencowleg,zephyrtronium"

	PREFIX = 2
	DICT   = "markov.2.dict"

	GenerateLength = 375
)

var (
	prefix   int
	complete uint32
	sending  uint32
	hasher   = sha1.New()

	TIMEOUT = 300 * time.Second

	limiter = rate.NewLimiter(rate.Every(2050*time.Millisecond), 1)

	rng     crazy.RNG
	uniform crazy.Uniform0_1

	queueAdd  = regexp.MustCompile(`(?i)(?P<please1>please,?\s+)?add\s+(?P<code>(?-i)[A-Z0-9]{4}-[A-Z0-9]{4})(?:\s+to\s+the\s+queue,?)?(?P<please2>\s+please)?`)
	queueNum  = regexp.MustCompile(`(?i)how\s+many\s+levels`)
	queueNext = regexp.MustCompile(`(?i)(?P<op>next|random)(?:\s+level,?)?(?:\s+please)?`)
	queueBan  = regexp.MustCompile(`(?i)(?P<op>ban|unban)\s+(\P{S}+)`)
	queueOpen = regexp.MustCompile(`(?i)(?P<op>open|close)(?:\s+the\s+queue)?`)
	queueMax  = regexp.MustCompile(`(?i)(?P<op>max(?:\s+per\s+user)?)\s+(?P<n>\d+|inf(?:inity)?)`)
)

func Filter(c map[string][]string, words []string) {
	for i := 0; i < prefix; i++ {
		word := strings.Repeat("\x01 ", prefix-i) + strings.ToLower(strings.Join(words[0:i], " "))
		if i >= len(words) {
			c[word] = append(c[word], "\x00")
			return
		}
		c[word] = append(c[word], words[i])
	}
	for i := prefix; i < len(words); i++ {
		if len(words[i]) == 0 {
			if i < len(words)-1 {
				Filter(c, words[i+1:])
				return
			}
			break
		}
		word := strings.ToLower(strings.Join(words[i-prefix:i], " "))
		c[word] = append(c[word], words[i])
	}
	word := strings.ToLower(strings.Join(words[len(words)-prefix:], " "))
	c[word] = append(c[word], "\x00")
}

func Walk(c map[string][]string, word string) string {
	s := make([]string, 0, 20)
	sum := 0
	for sum < GenerateLength {
		words := c[strings.ToLower(word)]
		if words == nil {
			break
		}
		nextword := words[rng.Intn(len(words))]
		if nextword == "\x00" {
			break
		}
		if nextword != "\x01" {
			sum += len(nextword) + 1
			s = append(s, nextword)
		}
		word = strings.Join(append(strings.Fields(strings.TrimRight(word, " "))[1:], nextword), " ")
	}
	return strings.Join(s, " ")
}

type Brain struct {
	// queue is the queue of messages to learn.
	queue []string
	// chain is the Markov chain.
	chain map[string][]string
	// prefix is the prefix length.
	prefix int
}

func (b *Brain) Learn(msg string) {
	if cap(b.queue) == 0 {
		b.filter(msg)
		return
	}
	if len(b.queue) == cap(b.queue) {
		b.filter(b.queue[0])
		copy(b.queue, b.queue[1:])
		b.queue = b.queue[:len(b.queue)-1]
	}
	b.queue = append(b.queue, msg)
}

func (b *Brain) Marshal() ([]byte, error) {
	for _, msg := range b.queue {
		b.filter(msg)
	}
	b.queue = b.queue[:0]
	return json.MarshalIndent(b.chain, "", "\t")
}

func (b *Brain) Say() string {
	word := strings.Repeat("\x01 ", b.prefix)
	return Walk(b.chain, word)
}

func (b *Brain) Clear(sender string) int {
	n := 0
	for i, line := range b.queue {
		if len(line) == 0 || line[0] != ':' {
			continue
		}
		line = line[1:]
		k := strings.IndexByte(line, ' ')
		if k < 0 {
			continue
		}
		j := strings.IndexByte(line, '!')
		if j < 0 {
			j = k
		}
		if line[:j] != sender {
			if n == 0 {
				continue
			}
			b.queue[i-n] = line
		} else {
			n++
		}
		if i+n >= len(b.queue) {
			break
		}
	}
	b.queue = b.queue[:len(b.queue)-n]
	return n
}

func (b *Brain) Forget(search *regexp.Regexp) int {
	n := 0
	for i, line := range b.queue {
		if len(line) == 0 || line[0] != ':' {
			continue
		}
		line = line[1:]
		k := strings.IndexByte(line, ':')
		if k < 0 {
			continue
		}
		line = line[k+1:]
		if !search.MatchString(line) {
			if n == 0 {
				continue
			}
			b.queue[i-n] = line
		} else {
			n++
		}
		if i+n >= len(b.queue) {
			break
		}
	}
	b.queue = b.queue[:len(b.queue)-n]
	return n
}

func (b *Brain) SetRoll(n int) {
	switch {
	case cap(b.queue) > n:
		for i := 0; i < cap(b.queue)-n; i++ {
			b.filter(b.queue[i])
		}
		q := make([]string, n)
		copy(q, b.queue[cap(b.queue)-n:])
		b.queue = q
	case cap(b.queue) < n:
		q := make([]string, len(b.queue), n)
		copy(q, b.queue)
		b.queue = q
	}
}

func (b *Brain) filter(msg string) {
	stuff := strings.Fields(msg)
	if len(stuff) < 4 {
		panic("unexpectedly short message: " + msg)
	}
	if stuff[1] != "PRIVMSG" {
		panic("unexpected message type " + stuff[1] + ": " + msg)
	}
	words := stuff[3:]
	words[0] = words[0][1:]
	Filter(b.chain, words)
}

type viewerLevel struct {
	code      string
	submitter string
}

type levelQueue struct {
	current *viewerLevel
	queue   []viewerLevel
	open    bool
	banned  map[string]bool
	perUser int
	max     int
}

func (q *levelQueue) add(code, submitter string) error {
	code = strings.ToUpper(code)
	if !q.open {
		return fmt.Errorf("the queue is closed")
	}
	if q.max > 0 && len(q.queue) >= q.max {
		return fmt.Errorf("the queue is full")
	}
	if q.banned[submitter] {
		return fmt.Errorf("you're banned lolmao")
	}
	if n := q.check(submitter); q.perUser > 0 && len(n) >= q.perUser {
		if q.perUser == 1 {
			return fmt.Errorf("you already have a level in the queue")
		}
		return fmt.Errorf("you already have %d levels in the queue", len(n))
	}
	for _, l := range q.queue {
		if l.code == code {
			return fmt.Errorf("%v is already in the queue", code)
		}
	}
	q.queue = append(q.queue, viewerLevel{code: code, submitter: submitter})
	return nil
}

func (q *levelQueue) next(random bool) (*viewerLevel, error) {
	if len(q.queue) == 0 {
		return nil, fmt.Errorf("the queue is empty")
	}
	k := 0
	if random {
		k = rng.Intn(len(q.queue))
	}
	q.current = &q.queue[k]
	copy(q.queue[k:], q.queue[k+1:])
	q.queue = q.queue[:len(q.queue)-1]
	return q.current, nil
}

func (q *levelQueue) check(submitter string) (r []string) {
	if q.current != nil && q.current.submitter == submitter {
		r = append(r, q.current.code)
	}
	for _, l := range q.queue {
		if l.submitter == submitter {
			r = append(r, l.code)
		}
	}
	return r
}

func (q *levelQueue) ban(submitter string) {
	if q.banned == nil {
		q.banned = make(map[string]bool)
	}
	q.banned[submitter] = true
	n := make([]viewerLevel, 0, len(q.queue))
	for _, l := range q.queue {
		if l.submitter != submitter {
			n = append(n, l)
		}
	}
	q.queue = n
}

func (q *levelQueue) unban(submitter string) {
	delete(q.banned, submitter)
}

func fail(args ...interface{}) {
	log.Println(args...)
	atomic.StoreUint32(&complete, 1)
}

func sender(send <-chan string, f net.Conn) {
	atomic.StoreUint32(&sending, 1)
	buf := make([]byte, 512)
	for atomic.LoadUint32(&complete) == 0 {
		select {
		case msg := <-send:
			if len(msg) > 450 {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
			err := limiter.Wait(ctx)
			cancel()
			if err != nil {
				log.Println("rate limiter wait error:", err)
				continue
			}
			if !strings.HasPrefix(msg, "PONG") {
				log.Println(msg)
			}
			copy(buf, msg)
			copy(buf[len(msg):], "\r\n")
			f.SetWriteDeadline(time.Now().Add(TIMEOUT))
			_, err = f.Write(buf[:len(msg)+2])
			switch e := err.(type) {
			case nil: // do nothing
			case net.Error:
				fail("net error while sending:", e)
			default:
				fail("error while sending:", err)
			}
		case <-time.After(TIMEOUT): // do nothing
		}
	}
	atomic.StoreUint32(&sending, 0)
}

func hash(word string) string {
	defer hasher.Reset()
	io.WriteString(hasher, word)
	return string(hasher.Sum(nil))
}

func recver(recv chan<- string, f net.Conn) {
	b := bufio.NewReader(f)
	cache := strings.Builder{}
	for atomic.LoadUint32(&complete) == 0 {
		f.SetReadDeadline(time.Now().Add(TIMEOUT))
		data, isPrefix, err := b.ReadLine()
		if len(data) > 0 {
			cache.Write(data)
		}
		switch e := err.(type) {
		case nil: // do nothing
		case net.Error:
			if e.Temporary() && !e.Timeout() {
				log.Println("temporary net error while recving:", e)
				break
			}
			fail("net error while recving:", e)
		default:
			fail("error while sending:", err)
		}
		if isPrefix {
			continue
		}
		if cache.Len() > 0 {
			line := cache.String()
			if line[0] == '@' {
				// trim off tags
				i := strings.Index(line, " ")
				// log.Println("tags:", line[:i])
				line = line[i+1:]
			}
			recv <- line
			cache.Reset()
		}
	}
	close(recv)
}

func talk(send chan<- string, meta, msg string, speed int) {
	time.Sleep(time.Millisecond * time.Duration(len(msg)*speed))
	send <- meta + msg + lennie()
}

func main() {
	var server, pass, nick, user, real, channel, listen, dict, secret, ssp, ign, ri, adm string
	var sendprob float64
	var caps, respond, dins bool
	var speed, roll int
	flag.StringVar(&server, "server", Server, "server and port to which to connect")
	flag.StringVar(&pass, "pass", "", "server login password")
	flag.StringVar(&nick, "nick", Nick, "nickname to use")
	flag.StringVar(&user, "user", User, "username to use")
	flag.StringVar(&real, "real", Real, "realname to use")
	flag.StringVar(&channel, "channel", Channel, "(comma-separated list of) channel(s) to join and in which to speak")
	flag.StringVar(&listen, "listen", Listen, "(comma-separated list of) channel(s) to join and in which to listen")
	flag.IntVar(&prefix, "length", PREFIX, "length of markov chain prefixes")
	flag.StringVar(&dict, "dict", DICT, "chain serialization file")
	flag.StringVar(&secret, "secret", "", "password for commands; unavailable by default")
	flag.Float64Var(&sendprob, "sendprob", 0.2, "default probability of responding")
	flag.StringVar(&ssp, "ssp", "", "special sendprobs, comma-sep list of chan=p")
	flag.BoolVar(&respond, "respond", true, "guarantee response when first word contains the bot's nick")
	flag.BoolVar(&caps, "caps", false, "send CAP REQ messages for twitch extensions")
	flag.StringVar(&ign, "ignore", Ignore, "comma-sep list of users from whom not to learn")
	flag.StringVar(&ri, "regexignore", RegexIgnore, "regular expression for PRIVMSGs to ignore")
	flag.StringVar(&adm, "admin", Admins, "comma-sep list of users from whom to accept cmds")
	flag.IntVar(&speed, "speed", 80, "\"typing\" speed in ms/char")
	flag.IntVar(&roll, "roll", 0, "number of messages to delay learning")
	flag.BoolVar(&dins, "dins", false, "ask what was for dins")
	flag.Parse()
	secret = hash(":" + secret)
	if prefix < 1 {
		log.Fatalln("prefix must be a positive integer")
	}
	ignored := make(map[string]bool)
	if len(ign) > 0 {
		for _, name := range strings.Split(strings.ToLower(ign), ",") {
			ignored[name] = true
		}
	}
	log.Printf("filter expression: %q\n", ri)
	re, err := regexp.Compile(ri)
	if err != nil {
		log.Println("error compiling regexignore:", err)
		log.Println("##############################################")
		log.Println("##        !!!no message filtering!!!        ##")
		log.Println("##############################################")
	}
	admins := make(map[string]bool)
	if len(adm) > 0 {
		for _, name := range strings.Split(strings.ToLower(adm), ",") {
			admins[name] = true
		}
	}
	mt := &crazy.MT64{}
	crazy.CryptoSeeded(mt, 8)
	rng = crazy.RNG{mt}
	uniform = crazy.Uniform0_1{mt}
	brain := Brain{queue: make([]string, 0, roll), prefix: prefix}
	if j, err := ioutil.ReadFile(dict); err != nil {
		log.Println("unable to open", dict+":", err)
		brain.chain = make(map[string][]string)
	} else if err = json.Unmarshal(j, &brain.chain); err != nil {
		log.Println("failed to unmarshal from", dict+":", err)
		brain.chain = make(map[string][]string)
	}
	levelqs := make(map[string]*levelQueue)
	chanset := make(map[string]float64)
	for _, c := range strings.Split(channel, ",") {
		chanset[c] = sendprob
	}
	if len(ssp) > 0 {
		for _, cv := range strings.Split(ssp, ",") {
			s := strings.Split(cv, "=")
			if v, err := strconv.ParseFloat(s[1], 64); err == nil {
				chanset[s[0]] = v
			} else {
				log.Println("malformed ssp on", cv)
				continue
			}
		}
	}
	if dins {
		memes := []string{
			"btw you should stretch, hydrate, and take care of yourself <3",
			"desu",
			"desu",
		}
		lennies = append(lennies, memes...)
	}
	addr, err := net.ResolveTCPAddr("tcp", server)
	if err != nil {
		log.Fatalln("error resolving", server+":", err)
	}
	sock, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Fatalln("error connecting to", server+":", err)
	}
	if err := sock.SetKeepAlive(false); err != nil {
		log.Println("unable to disable keep-alive:", err)
	}
	sock.SetWriteDeadline(time.Now().Add(TIMEOUT))
	send := make(chan string)
	recv := make(chan string)
	go sender(send, sock)
	go recver(recv, sock)
	req := make([]string, 0, 3)
	if roll > 0 {
		req = append(req, "twitch.tv/commands")
	}
	if caps {
		if len(req) == 0 {
			req = append(req, "twitch.tv/commands")
		}
		req = append(req, "twitch.tv/tags", "twitch.tv/membership")
	}
	if len(req) > 0 {
		send <- "CAP REQ :" + strings.Join(req, " ")
	}
	if pass != "" {
		send <- "PASS " + pass
	}
	send <- "NICK " + nick
	send <- "USER " + user + " * * :" + real
	end := func() {
		if j, err := brain.Marshal(); err != nil {
			log.Println("failed to marshal dict:", err)
			return
		} else if err = ioutil.WriteFile(dict, j, 0644); err != nil {
			log.Println("failed to marshal into", dict+":", err)
			return
		} else {
			send <- "QUIT :goodbye"
		}
		atomic.StoreUint32(&complete, 1)
		time.AfterFunc(5*time.Second, func() { os.Exit(0) })
	}
	isig := make(chan os.Signal, 3)
	ksig := make(chan os.Signal, 3)
	signal.Notify(isig, os.Interrupt)
	signal.Notify(ksig, os.Kill)
	for atomic.LoadUint32(&sending) != 0 {
		select {
		case line, ok := <-recv:
			if !ok {
				break
			}
			stuff := strings.Fields(line)
			if stuff[0] == "PING" {
				send <- "PONG " + strings.Join(stuff[1:], " ")
			} else {
				log.Println(line)
				if len(stuff) > 1 {
					switch stuff[1] {
					case "376":
						send <- "JOIN " + channel
						send <- "JOIN " + listen
					case "PRIVMSG":
						from := strings.ToLower(stuff[0][1:strings.IndexAny(stuff[0], "! ")])
						if stuff[2] == nick || (stuff[2] == "#"+strings.ToLower(nick) && admins[from]) {
							if l := len(stuff); l > 5 && hash(stuff[3]) == secret {
								switch strings.ToLower(stuff[4]) {
								case "quit":
									send <- "QUIT :" + strings.Join(stuff[5:], " ")
									end()
								case "join":
									for _, c := range stuff[5:] {
										chanset[c] = sendprob
										send <- "JOIN " + c
									}
								case "listen":
									for _, c := range stuff[5:] {
										chanset[c] = 0
										send <- "JOIN " + c
									}
								case "part":
									for _, c := range stuff[5:] {
										delete(chanset, c)
										send <- "PART " + c
									}
								case "nick":
									send <- "NICK " + stuff[5]
								case "sendprob":
									if v, err := strconv.ParseFloat(stuff[5], 64); err == nil {
										if l > 6 {
											for _, c := range stuff[6:] {
												chanset[c] = v
											}
										} else {
											sendprob = v
											log.Println("send probability", sendprob)
											for c, p := range chanset {
												if p > 0 {
													chanset[c] = sendprob
												}
											}
										}
									}
								case "raw":
									send <- strings.Join(stuff[5:], " ")
								case "ignore":
									for _, c := range stuff[5:] {
										ignored[c] = true
									}
									o := make([]string, 0, len(ignored))
									for a, ok := range ignored {
										if ok {
											o = append(o, a)
										}
									}
									log.Println("ignored:", o)
								case "unignore":
									for _, c := range stuff[5:] {
										ignored[c] = false
									}
									o := make([]string, 0, len(ignored))
									for a, ok := range ignored {
										if ok {
											o = append(o, a)
										}
									}
									log.Println("ignored:", o)
								case "admin":
									for _, c := range stuff[5:] {
										admins[c] = true
									}
									o := make([]string, 0, len(admins))
									for a := range admins {
										o = append(o, a)
									}
									log.Println("admins:", o)
								case "respond":
									respond = strings.EqualFold(stuff[5], "on")
									log.Println("guaranteed response set to", respond)
								case "regexignore":
									ri = strings.Join(stuff[5:], "\\s+")
									log.Printf("filter expression: %q\n", ri)
									if re, err = regexp.Compile(ri); err != nil {
										log.Println("error compiling regexignore:", err)
										log.Println("no message filtering!")
									}
								case "speed":
									if v, err := strconv.ParseInt(stuff[5], 10, 32); err == nil && v >= 0 {
										speed = int(v)
										log.Println("typing speed", speed)
									}
								case "roll":
									if roll == 0 && !caps {
										send <- "PRIVMSG " + stuff[2] + " :I can't see CLEARCHAT commands, so the roll queue is disabled."
										continue
									}
									if v, err := strconv.ParseInt(stuff[5], 10, 32); err == nil && v >= 0 {
										brain.SetRoll(int(v))
										log.Println("roll length:", v)
									}
								case "viewerlevels":
									for _, c := range stuff[5:] {
										levelqs[c] = &levelQueue{}
									}
									log.Printf("levelqs: %#v\n", levelqs)
								case "forget":
									fg := strings.Join(stuff[5:], "\\s+")
									log.Printf("forget expression: %q\n", fg)
									fe, err := regexp.Compile(fg)
									if err != nil {
										send <- fmt.Sprintf("PRIVMSG %s :Failed to compile %q: %v", stuff[2], fg, err)
										continue
									}
									log.Println("forgot", brain.Forget(fe), "messages")
								case "status":
									cfg := statusconfigs[strings.ToLower(stuff[5])]
									if cfg.nick {
										send <- fmt.Sprintf("PRIVMSG %s :I believe I am %s.", stuff[2], nick)
									}
									if cfg.req {
										if len(req) != 0 {
											send <- fmt.Sprintf("PRIVMSG %s :Twitch capabilities I've requested: %v", stuff[2], req)
										} else {
											send <- fmt.Sprintf("PRIVMSG %s :I haven't requested any Twitch capabilities.", stuff[2])
										}
									}
									o := []string{}
									if cfg.admin {
										for a := range admins {
											o = append(o, a)
										}
										send <- fmt.Sprintf("PRIVMSG %s :Admins: %s", stuff[2], strings.Join(o, ", "))
										o = o[:0]
									}
									if cfg.ignored {
										for a := range ignored {
											o = append(o, a)
										}
										send <- fmt.Sprintf("PRIVMSG %s :Ignored: %s", stuff[2], strings.Join(o, ", "))
										o = o[:0]
									}
									if cfg.chans {
										for c, p := range chanset {
											o = append(o, fmt.Sprintf("%s: %g%%", c, p*100))
										}
										send <- fmt.Sprintf("PRIVMSG %s :Channels with send probabilities: %s", stuff[2], strings.Join(o, ", "))
									}
									if cfg.re {
										send <- fmt.Sprintf("PRIVMSG %s :I ignore messages matching this regular expression: %v", stuff[2], re)
									}
									if cfg.respond {
										if respond {
											send <- fmt.Sprintf("PRIVMSG %s :I respond to messages directed at me.", stuff[2])
										} else {
											send <- fmt.Sprintf("PRIVMSG %s :I do not respond to messages directed to me.", stuff[2])
										}
									}
									if cfg.roll {
										if cap(brain.queue) == 0 {
											send <- fmt.Sprintf("PRIVMSG %s :I do not wait before learning messages.", stuff[2])
										} else {
											send <- fmt.Sprintf("PRIVMSG %s :I wait for %d messages before learning, with %d currently pending.", stuff[2], cap(brain.queue), len(brain.queue))
										}
									}
									if cfg.speed {
										send <- fmt.Sprintf("PRIVMSG %s :I type at a rate of %d ms/char = %d char/s.", stuff[2], speed, 1000/speed)
									}
									if cfg.knowledge {
										nw := 0
										uw := 0
										aw := make(map[string]bool)
										for _, w := range brain.chain {
											mm := make(map[string]bool)
											nw += len(w)
											for _, word := range w {
												if mm[word] {
													continue
												}
												mm[word] = true
												uw++
												aw[word] = true
											}
										}
										send <- fmt.Sprintf("PRIVMSG %s :I know %d prefixes of length %d with %d total suffixes, %d of which are unique per prefix, and %d of which are unique overall. This means a %.2f:1 learning ratio and a %.2f%% uniqueness index.", stuff[2], len(brain.chain), brain.prefix, nw, uw, len(aw), float64(nw)/float64(len(brain.chain)), float64(uw)*100/float64(nw))
									}
								default:
									goto thisisanokuseofgotoiswear
								}
								break
							}
						}
					thisisanokuseofgotoiswear:
						if ignored[from] {
							log.Println(from, "is ignored")
							break
						}
						words := stuff[3:]
						words[0] = words[0][1:]
						addressed := strings.Contains(strings.ToLower(words[0]), strings.ToLower(nick))
						if addressed {
							log.Println("someone is talking to me")
						}
						if q := levelqs[stuff[2]]; addressed && q != nil {
							msgline := strings.Join(words, " ")
							if m := queueAdd.FindStringSubmatch(msgline); m != nil {
								if m[1] == "" && m[3] == "" {
									talk(send, "PRIVMSG "+stuff[2]+" :", "be polite", 0)
									break
								}
								if err := q.add(m[2], from); err != nil {
									talk(send, "PRIVMSG "+stuff[2]+" :", "@"+from+" "+err.Error(), 0)
									break
								}
								talk(send, "PRIVMSG "+stuff[2]+" :", fmt.Sprintf("@%s added %v - there are now %d levels waiting in the queue", from, m[2], len(q.queue)), 0)
								break
							}
							if admins[from] {
								if m := queueNext.FindStringSubmatch(msgline); m != nil {
									level, err := q.next(m[1] == "random")
									if err != nil {
										talk(send, "PRIVMSG "+stuff[2]+" :", "Couldn't get a level because "+err.Error(), 0)
										break
									}
									talk(send, "PRIVMSG "+stuff[2]+" :", fmt.Sprintf("Next level is %v from @%v", level.code, level.submitter), 0)
									break
								}
								if m := queueBan.FindStringSubmatch(msgline); m != nil {
									if m[1] == "ban" {
										q.ban(m[2])
									} else {
										q.unban(m[2])
									}
									break
								}
								if m := queueOpen.FindStringSubmatch(msgline); m != nil {
									q.open = m[1] == "open"
									break
								}
								if m := queueMax.FindStringSubmatch(msgline); m != nil {
									var n int
									if strings.HasPrefix(strings.ToLower(m[2]), "inf") {
										n = 0
									} else {
										k, err := strconv.ParseInt(m[2], 10, 32)
										if err != nil {
											talk(send, "PRIVMSG "+stuff[2]+" :", err.Error(), 0)
											break
										}
										n = int(k)
									}
									if m[1] == "max" {
										q.max = n
									} else {
										q.perUser = n
									}
									break
								}
								if queueNum.MatchString(msgline) {
									if len(q.queue) == 1 {
										talk(send, "PRIVMSG "+stuff[2]+" :", "There is one more level in the qUwUe", 0)
										break
									}
									talk(send, "PRIVMSG "+stuff[2]+" :", fmt.Sprintf("There are %d more levels in the queue", len(q.queue)), 0)
									break
								}
							}
						}
						addressed = addressed && respond
						if !addressed && re != nil && re.MatchString(strings.Join(words, " ")) {
							log.Println("filtered out message")
							break
						}
						if line[len(line)-1] != 1 { // drop ctcps
							if len(words) >= 1 {
								if !addressed {
									brain.Learn(line)
								}
								if addressed || uniform.Next() < chanset[stuff[2]] {
									wk := brain.Say()
									if badmatch(strings.Fields(wk), words) {
										log.Println("generated:", wk)
										break // drop unoriginal messages
									}
									if !addressed {
										talk(send, "PRIVMSG "+stuff[2]+" :", wk, speed)
									} else {
										talk(send, "PRIVMSG "+stuff[2]+" :", wk, speed/3)
									}
								}
							}
						}
					case "KICK":
						if stuff[3] == nick {
							send <- "JOIN " + stuff[2]
						}
					case "NICK":
						if strings.HasPrefix(stuff[0], ":"+nick) &&
							(line[len(nick)+1] == '!' || line[len(nick)+1] == ' ') {
							nick = stuff[2][1:]
							println("nick is " + nick)
						}
					case "CLEARCHAT":
						if len(stuff) == 4 {
							sender := stuff[3][1:]
							cleared := brain.Clear(sender)
							log.Println("cleared", cleared, "messages from", sender)
						}
					}
				}
			}
		case <-isig:
			end()
		case <-ksig:
			atomic.StoreUint32(&complete, 1)
			atomic.StoreUint32(&sending, 0)
			continue
		}
	}
}

var lennies = []string{
	"¯\\_( ͡° ͜ʖ ͡°)_/¯",
	"xD",
	"( ͡° ͜ʖ ͡°)",
	"(◕◡◕)",
	"( •̀︹•́)",
	"(/ω＼)",
	"(╭☞ ͠°ᗜ °)╭☞",
	"∠( ᐛ 」∠)＿",
	"(´；ω；｀)",
	";)",
	"PogChamp",
	"ლ(´ڡ`ლ)",
	"D:",
	"",
	"",
	"",
	"",
}

func lennie() string {
	return " " + lennies[rng.Intn(len(lennies))]
}

func badmatch(walk, src []string) (match bool) {
	if len(walk) > len(src) {
		return false
	}
	// it would be faster to start at the end and walk backward
	for i := len(src) - 1; i-(len(src)-len(walk)) >= 0; i-- {
		word := walk[i-(len(src)-len(walk))]
		if strings.ToLower(src[i]) != strings.ToLower(word) {
			return false
		}
	}
	return true
}

type statusconfig struct {
	nick, req, admin, ignored, chans, re, respond, roll, speed, knowledge bool
}

var statusconfigs = map[string]statusconfig{
	"all":       {true, true, true, true, true, true, true, true, true, true},
	"nick":      {nick: true},
	"req":       {req: true},
	"admin":     {admin: true},
	"ignored":   {ignored: true},
	"chans":     {chans: true},
	"re":        {re: true},
	"respond":   {respond: true},
	"roll":      {roll: true},
	"speed":     {speed: true},
	"knowledge": {knowledge: true},
}
