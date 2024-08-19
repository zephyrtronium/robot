package command

import (
	"log/slog"
	"strings"
	"unicode"
)

// Effect applies an effect to a message.
// The currently available effects include "OwO", "AAAAA", "o", and "".
// Names are not case sensitive.
func Effect(name, msg string) string {
	var r string
	switch {
	case name == "":
		return msg
	case strings.EqualFold(name, "OwO"):
		r = owoize(msg)
	case strings.EqualFold(name, "AAAAA"):
		r = lenlimit(aaaaaize(msg), 40)
	case strings.EqualFold(name, "o"):
		r = oize(msg)
	default:
		slog.Error("no such effect", slog.String("name", name), slog.String("msg", msg))
		return msg
	}
	slog.Info("applied effect", slog.String("name", name), slog.String("in", msg), slog.String("out", r))
	return r
}

func lenlimit(msg string, lim int) string {
	if len(msg) > lim {
		r := []rune(msg)
		r = r[:min(len(r), lim)]
		msg = string(r)
	}
	return msg
}

func owoize(msg string) string {
	return owoRep.Replace(msg)
}

var owoRep = strings.NewReplacer(
	"r", "w", "R", "W",
	"l", "w", "L", "W",
	"na", "nya", "Na", "Nya", "NA", "NYA",
	"ni", "nyi", "Ni", "Nyi", "NI", "NYI",
	"nu", "nyu", "Nu", "Nyu", "NU", "NYU",
	"ne", "nye", "Ne", "Nye", "NE", "NYE",
	"no", "nyo", "No", "Nyo", "NO", "NYO",
)

func aaaaaize(msg string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return 'A'
		}
		return r
	}, msg)
}

func oize(msg string) string {
	return oRep.Replace(msg)
}

var oRep = strings.NewReplacer(
	"a", "o", "e", "o", "i", "o", "u", "o",
	"A", "O", "E", "O", "I", "O", "U", "O",
)
