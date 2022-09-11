package brain

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Tuple is a single Markov chain tuple.
type Tuple struct {
	Prefix []string
	Suffix string
}

// Learner records Markov chain tuples.
type Learner interface {
	// Order returns the number of elements in the prefix of a chain.
	Order() int
	// Learn records a set of tuples. Each tuple prefix has length equal to the
	// result of Order. Tuples should be stored without modification.
	Learn(ctx context.Context, tuples []Tuple)
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
