package brain

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/rangetable"
)

// Ranges we collect into terms.
var (
	ln   = rangetable.Merge(unicode.L, unicode.N, unicode.Pc, unicode.Pd)
	syms = rangetable.Merge(unicode.M, unicode.P, unicode.S)
)

// Tokens converts a message into a list of its words appended to dst.
func Tokens(dst []string, msg string) []string {
	start := len(dst)
	for len(msg) > 0 {
		// The general procedure is to find which of several sets of runes
		// the first character is in, continue accumulating until finding any
		// different set (including spaces), add the first following space if
		// there is one, and take the portion to that point.
		// Then, skip remaining spaces and repeat.
		c, l := utf8.DecodeRuneInString(msg)
		switch {
		case c == '@':
			// Since we're at the start of a token, treat this as a letter or
			// number so it combines with a subsequent username.
			// We could do more advanced things by looking ahead in the string
			// to verify we're looking at a name, but that is much more code to
			// write for a case that will be uncommon.
			// In terms of control flow, we can fall through to the next case,
			// after we skip past the @ itself.
			if l == len(msg) {
				break
			}
			l++
			fallthrough
		case unicode.Is(ln, c):
			for l < len(msg) {
				c, k := utf8.DecodeRuneInString(msg[l:])
				if !unicode.Is(ln, c) {
					break
				}
				l += k
			}
		case unicode.Is(syms, c):
			for l < len(msg) {
				c, k := utf8.DecodeRuneInString(msg[l:])
				if !unicode.Is(syms, c) {
					break
				}
				l += k
			}
		default:
			// Space, control code, or something eldritch.
			// Skip.
			msg = msg[l:]
			continue
		}
		// Note that we test only for U+0020 space, not unicode.IsSpace.
		if l < len(msg) && msg[l] == ' ' {
			l++
		}
		dst = append(dst, msg[:l])
		msg = msg[l:]
	}
	// Ensure the last token we added ends with a space.
	if len(dst) > start {
		w := dst[len(dst)-1]
		if len(w) > 0 && w[len(w)-1] != ' ' {
			dst[len(dst)-1] += " "
		}
	}
	return dst
}

// ReduceEntropy transforms a term in a way which makes it more likely to
// equal other terms transformed the same way.
func ReduceEntropy(w string) string {
	return strings.ToLower(w)
}
