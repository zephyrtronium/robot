package brain

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Tuple is a single Markov chain tuple.
type Tuple struct {
	Prefix []string
	Suffix string
}

// MessageMeta holds metadata about a message.
type MessageMeta struct {
	// Time is the time at which the message was sent.
	Time time.Time
	// ID is a unique ID for the message.
	ID string
	// Tag is a tag that should be associated with the message data.
	Tag string
	// User is an identifier for the user. It is obfuscated such that the user
	// cannot be identified and is not correlated between rooms.
	User [32]byte
}

// Learner records Markov chain tuples.
type Learner interface {
	// Order returns the number of elements in the prefix of a chain. It is
	// called once at the beginning of learning. The returned value must always
	// be at least 1.
	Order() int
	// Learn records a set of tuples. Each tuple prefix has length equal to the
	// result of Order. The tuples begin with empty strings in the prefix to
	// denote the start of the message and end with one empty suffix to denote
	// the end; all other tokens are non-empty. Each tuple's prefix has entropy
	// reduction transformations applied.
	Learn(ctx context.Context, meta *MessageMeta, tuples []Tuple) error
}

// Learn records tokens into a Learner.
func Learn(ctx context.Context, l Learner, meta *MessageMeta, toks []string) error {
	n := l.Order()
	if n < 1 {
		panic(fmt.Errorf("order must be at least 1, got %d from %#v", n, l))
	}
	tt := make([]Tuple, 0, len(toks)+1)
	p := Tuple{Prefix: make([]string, n)}
	for _, w := range toks {
		q := Tuple{Prefix: make([]string, n), Suffix: w}
		copy(q.Prefix, p.Prefix[1:])
		q.Prefix[n-1] = strings.ToLower(p.Suffix)
		tt = append(tt, q)
		p = q
	}
	q := Tuple{Prefix: make([]string, n), Suffix: ""}
	copy(q.Prefix, p.Prefix[1:])
	q.Prefix[n-1] = strings.ToLower(p.Suffix)
	tt = append(tt, q)
	return l.Learn(ctx, meta, tt)
}

// Tokens converts a message into a list of its words appended to dst.
func Tokens(dst []string, msg string) []string {
	art := false
	msg = strings.TrimSpace(msg)
	for msg != "" {
		k := strings.IndexFunc(msg, unicode.IsSpace)
		if k == 0 {
			_, n := utf8.DecodeRuneInString(msg)
			msg = msg[n:]
			continue
		}
		if k < 0 {
			k = len(msg)
		}
		word := msg[:k]
		// Some English-specific stuff: if word is an article (a, an, the), and
		// another word follows, then the token is both words together. To
		// implement this, we track whether the previous word was an article
		// and add to it if so. As a special case for the special case, "a"
		// might be part of D A N K M E M E S, so if the previous word was "a"
		// and the current one is length 1, then we do not join.
		if art {
			if utf8.RuneCountInString(word) != 1 || !strings.EqualFold(dst[len(dst)-1], "a") {
				dst[len(dst)-1] += " " + word
				art = false
				msg = msg[k:]
				continue
			}
		}
		dst = append(dst, word)
		art = isArticle(word)
		// Advance to k and then skip the next rune, since it is whitespace if
		// it exists. This might be the last word in msg, in which case
		// DecodeRuneInString will return 0 for the length.
		msg = msg[k:]
		_, n := utf8.DecodeRuneInString(msg)
		msg = msg[n:]
	}
	return dst
}

func isArticle(word string) bool {
	return strings.EqualFold(word, "a") || strings.EqualFold(word, "an") || strings.EqualFold(word, "the")
}
