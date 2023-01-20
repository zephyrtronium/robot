package channel

import (
	"regexp"

	"golang.org/x/time/rate"

	"github.com/zephyrtronium/robot/v2/distro"
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
	// Emotes is the distribution of emotes.
	Emotes *distro.Dist[string]
	// Effects is the distribution of effects.
	Effects *distro.Dist[string]
}

// SetEmotes forms the emote cdf from the union of the given emote maps.
func (c *Channel) SetEmotes(ms ...map[string]int) *Channel {
	if len(ms) == 0 {
		return c
	}
	u := make(map[string]int)
	for _, m := range ms {
		for k, v := range m {
			u[k] += v
		}
	}
	c.Emotes = distro.New(distro.FromMap(u))
	return c
}

// SetEffects forms the effect cdf from the union of the given effect maps.
func (c *Channel) SetEffects(ms ...map[string]int) *Channel {
	if len(ms) == 0 {
		return c
	}
	u := make(map[string]int)
	for _, m := range ms {
		for k, v := range m {
			u[k] += v
		}
	}
	c.Emotes = distro.New(distro.FromMap(u))
	return c
}
