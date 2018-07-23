package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
	// "unicode"
)

const (
	Server = "irc.twitch.tv:6667"
	//Pass    = "oauth:longgibberishgoeshere"
	Nick    = "robotisbroken"
	User    = "robotisbroken"
	Real    = "robotisbroken"
	Channel = "#brokencowleg,#robotisbroken"
	Listen  = ""
	Ignore  = "zbot,nightbot"
	Admins  = "brokencowleg,zephyrtronium"

	PREFIX = 2
	DICT   = "markov.2.dict"
)

var (
	prefix   int
	complete bool
	sending  bool
	hasher   = sha1.New()

	TIMEOUT = 300 * time.Second
)

func Filter(c map[string][]string, words []string) {
	if len(words) < prefix {
		return
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
	word := strings.ToLower(strings.Join(words[len(words)-prefix:len(words)], " "))
	c[word] = append(c[word], "\x00")
}

func Walk(c map[string][]string, word string) string {
	s := make([]string, 0, 20)
	s = append(s, word)
	sum := 0
	for sum < 400 {
		words := c[strings.ToLower(word)]
		if words == nil {
			break
		}
		nextword := words[rand.Intn(len(words))]
		if nextword == "\x00" {
			break
		}
		sum += len(nextword) + 1
		s = append(s, nextword)
		word = strings.Join(append(strings.Split(word, " ")[1:], nextword), " ")
	}
	return strings.Join(s, " ")
}

func sender(send <-chan string, f net.Conn) {
	sending = true
	t := time.Now().UnixNano()
	buf := make([]byte, 512)
	for !complete {
		msg := <-send
		if len(msg) > 450 {
			continue
		}
		if t < time.Now().UnixNano() {
			t = time.Now().UnixNano()
		} else if t > time.Now().UnixNano()+7e9 {
			time.Sleep(2 * time.Second)
		}
		if !strings.HasPrefix(msg, "PONG") {
			log.Println(msg)
		}
		copy(buf, msg)
		copy(buf[len(msg):], "\r\n")
		f.SetWriteDeadline(time.Now().Add(TIMEOUT))
		_, err := f.Write(buf[:len(msg)+2])
		switch e := err.(type) {
		case nil: // do nothing
		case net.Error:
			log.Fatalln("net error while sending:", e)
			if e.Temporary() {
				continue
			}
		default:
			log.Fatalln("error while sending:", err)
		}
	}
	sending = false
}

func hash(word string) string {
	defer hasher.Reset()
	io.WriteString(hasher, word)
	return string(hasher.Sum(nil))
}

func recver(recv chan<- string, f net.Conn) {
	b := bufio.NewReader(f)
	cache := ""
	for !complete {
		f.SetReadDeadline(time.Now().Add(TIMEOUT))
		data, isPrefix, err := b.ReadLine()
		if len(data) > 0 {
			cache += string(data)
			switch e := err.(type) {
			case nil: // do nothing
			case net.Error:
				if e.Temporary() {
					break
				}
				log.Fatalln("net error while recving:", e)
			default:
				log.Fatalln("error while sending:", err)
			}
			if isPrefix {
				continue
			}
			if cache[0] == '@' {
				// trim off tags
				i := strings.Index(cache, " ")
				// log.Println("tags:", cache[:i])
				cache = cache[i+1:]
			}
			recv <- cache
			cache = ""
		}
	}
	close(recv)
}

func main() {
	var server, pass, nick, user, real, channel, listen, dict, secret, ign, adm string
	var sendprob float64
	var caps bool
	flag.StringVar(&server, "server", Server, "server and port to which to connect")
	flag.StringVar(&pass, "pass", "", "server login password")
	flag.StringVar(&nick, "nick", Nick, "nickname to use")
	flag.StringVar(&user, "user", User, "username to use")
	flag.StringVar(&real, "real", Real, "realname to use")
	flag.StringVar(&channel, "channel", Channel,
		"(comma-separated list of) channel(s) to join and in which to speak")
	flag.StringVar(&listen, "listen", Listen,
		"(comma-separated list of) channel(s) to join and in which to listen")
	flag.IntVar(&prefix, "length", PREFIX, "length of markov chain prefixes")
	flag.StringVar(&dict, "dict", DICT, "chain serialization file")
	flag.StringVar(&secret, "secret", "", "password for commands; unavailable by default")
	flag.Float64Var(&sendprob, "sendprob", 0.2, "probability of responding")
	flag.BoolVar(&caps, "caps", false, "send CAP REQ messages for twitch extensions")
	flag.StringVar(&ign, "ignore", Ignore, "comma-sep list of users from whom not to learn")
	flag.StringVar(&adm, "admin", Admins, "comma-sep list of users from whom to accept cmds")
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
	admins := make(map[string]bool)
	if len(adm) > 0 {
		for _, name := range strings.Split(strings.ToLower(adm), ",") {
			admins[name] = true
		}
	}
	rand.Seed(time.Now().UnixNano())
	var chain map[string][]string
	if j, err := ioutil.ReadFile(dict); err != nil {
		chain = make(map[string][]string)
	} else if err = json.Unmarshal(j, &chain); err != nil {
		log.Println("failed to unmarshal from", dict+":", err)
		chain = make(map[string][]string)
	}
	chanset := make(map[string]bool)
	for _, c := range strings.Split(channel, ",") {
		chanset[c] = true
	}
	addr, err := net.ResolveTCPAddr("tcp", server)
	if err != nil {
		log.Fatalln("error resolving", server+":", err)
	}
	sock, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Fatalln("error connecting to", server+":", err)
	}
	sock.SetWriteDeadline(time.Now().Add(TIMEOUT))
	send := make(chan string)
	recv := make(chan string)
	go sender(send, sock)
	go recver(recv, sock)
	if caps {
		send <- "CAP REQ :twitch.tv/membership twitch.tv/commands twitch.tv/tags"
	}
	if pass != "" {
		send <- "PASS " + pass
	}
	send <- "NICK " + nick
	send <- "USER " + user + " * * :" + real
	end := func() {
		if j, err := json.Marshal(chain); err != nil {
			log.Println("failed to marshal dict:", err)
			return
		} else if err = ioutil.WriteFile(dict, j, 0644); err != nil {
			log.Println("failed to marshal into", dict+":", err)
			return
		} else {
			send <- "QUIT :goodbye"
		}
		complete = true
		time.AfterFunc(5*time.Second, func() { os.Exit(0) })
	}
	// WINDOWS-DEPENDENT:
	isig := make(chan os.Signal, 3)
	ksig := make(chan os.Signal, 3)
	signal.Notify(isig, os.Interrupt)
	signal.Notify(ksig, os.Kill)
	for sending {
		select {
		case line, ok := <-recv:
			if !ok {
				break
			}
			stuff := strings.Split(line, " ")
			if stuff[0] == "PING" {
				send <- "PONG " + strings.Join(stuff[1:], " ")
			} else {
				log.Println(line)
				if len(stuff) > 1 {
					// out:
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
										chanset[c] = true
										send <- "JOIN " + c
									}
								case "listen":
									for _, c := range stuff[5:] {
										chanset[c] = false
										send <- "JOIN " + c
									}
								case "part":
									for _, c := range stuff[5:] {
										chanset[c] = false
										send <- "PART " + c
									}
								case "nick":
									send <- "NICK " + stuff[5]
								case "sendprob":
									if v, err := strconv.ParseFloat(stuff[5], 64); err == nil {
										sendprob = v
										log.Println("send probability", sendprob)
									}
								case "raw":
									send <- strings.Join(stuff[5:], " ")
								case "ignore":
									for _, c := range stuff[5:] {
										ignored[c] = true
									}
								case "unignore":
									for _, c := range stuff[5:] {
										ignored[c] = false
									}
								case "admin":
									for _, c := range stuff[5:] {
										admins[c] = true
									}
								}
							}
							break
						}
						if ignored[from] {
							break
						}
						words := stuff[3:]
						words[0] = words[0][1:]
						// for i, word := range words {
						// 	words[i] = strings.TrimFunc(word,
						// 		func(r rune) bool {
						// 			return !(unicode.IsLetter(r) || unicode.IsDigit(r))
						// 		})
						// 	// if len(words[i]) == 1 && !strings.ContainsAny(words[i], "AIaioruy") {
						// 	// 	break out
						// 	// }
						// }
						if line[len(line)-1] != 1 { // drop ctcps
							if strings.Contains(line, "://") || strings.Contains(line, ".com/") {
								break // drop urls
							}
							if len(words) > prefix {
								Filter(chain, words)
								if chanset[stuff[2]] && rand.Float64() < sendprob {
									// r := rand.Intn(len(words) - prefix)
									r := 0
									word := strings.Join(words[r:r+prefix], " ")
									wk := Walk(chain, word)
									if badmatch(strings.Fields(wk), words) {
										log.Println("generated:", wk)
										break // drop unoriginal messages
									}
									send <- "PRIVMSG " + stuff[2] + " :" + wk + lennie()
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
					}
				}
			}
		case <-isig:
			end()
		case <-ksig:
			complete = true
			sending = false
			continue
			// case isig := <-signal.Incoming:
			// 	if usig, ok := sig.(os.UnixSignal); ok && usig == os.SIGINT {
			// 		end()
			// 	} else if ok && usig == os.SIGTERM {
			// 		complete = true
			// 		sending = false
			// 		continue
			// 	}
		}
	}
}

var lennies = []string{
	"¯\\_( ͡° ͜ʖ ͡°)_/¯",
	"xD",
	"( ͡° ͜ʖ ͡°)",
	"(◕◡◕)",
	"(∩⊜◡⊜)⊃━☆ﾟ.*",
	"(╯°□°）╯︵ ┻━┻",
	"(/ω＼)",
	"(╭☞ ͠°ᗜ °)╭☞",
	"∠( ᐛ 」∠)＿",
	"(´；ω；｀)",
	"",
}

func lennie() string {
	return " " + lennies[rand.Intn(len(lennies))]
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
