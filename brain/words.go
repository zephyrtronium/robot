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

package brain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Tokens converts a message into a list of its words.
func Tokens(msg string) []string {
	var r []string
	art := false
	msg = strings.TrimSpace(msg)
	for msg != "" {
		k := strings.IndexFunc(msg, unicode.IsSpace)
		if k == 0 {
			msg = msg[1:]
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
			if utf8.RuneCountInString(word) != 1 || !strings.EqualFold(r[len(r)-1], "a") {
				r[len(r)-1] += " " + word
				art = isArticle(word)
				msg = msg[k:]
				continue
			}
		}
		r = append(r, word)
		art = isArticle(word)
		// Note we only advance to k, not beyond, because if this is the last
		// word in the message, then msg[k+1:] would panic.
		msg = msg[k:]
	}
	return r
}

func isArticle(word string) bool {
	return strings.EqualFold(word, "a") || strings.EqualFold(word, "an") || strings.EqualFold(word, "the")
}
