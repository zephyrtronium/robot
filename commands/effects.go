package commands

import (
	"strings"
	"unicode"
)

// Effect applies a named effect to a message.
func Effect(effect, msg string) string {
	f := effects[effect]
	if f == nil {
		return msg
	}
	return f(msg)
}

var effects map[string]func(string) string

func init() {
	effects = map[string]func(string) string{
		"uwu":   uwuEffect,
		"AAAAA": aaaaaEffect,
		"me":    meEffect,
	}
}

func uwuEffect(s string) string {
	return uwuRep.Replace(s)
}

func aaaaaEffect(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return 'A'
		}
		return r
	}, s)
}

func meEffect(s string) string {
	return "\x01ACTION " + s + "\x01"
}
