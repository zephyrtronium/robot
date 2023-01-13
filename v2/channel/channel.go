package channel

import (
	"regexp"

	"golang.org/x/time/rate"
)

type Channel struct {
	// Name is the name of the channel.
	Name string
	// Learn and Send are the channel tags.
	Learn, Send string
	// Block is a regex that matches messages which should not be used for
	// learning.
	Block *regexp.Regexp
	// Responses is the probability that a received message will trigger a
	// random response.
	Responses float64
	// Rate is the rate limiter for messages. Attempts to speak in excess of
	// the rate limit are dropped.
	Rate *rate.Limiter
	// Memery is the meme detector for the channel.
	Memery *MemeDetector
	// EmoteWeight is the cdf of emotes for the channel.
	EmoteWeight []uint64
	// Emotes is the list of emotes, such that the cdf of Emotes[i] is
	// EmoteWeight[i].
	Emotes []string
}
