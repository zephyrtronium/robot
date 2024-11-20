package command

import (
	"log/slog"
	"math/rand/v2"
	"strings"
	"unicode"

	"gitlab.com/zephyrtronium/pick"
)

// Effect applies an effect to a message.
// The currently available effects include "OwO", "AAAAA", "o", "hte", and "".
// Names are not case sensitive.
func Effect(log *slog.Logger, name, msg string) string {
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
	case strings.EqualFold(name, "hte"):
		r = hteize(msg)
	default:
		log.Error("no such effect", slog.String("name", name), slog.String("msg", msg))
		return msg
	}
	log.Info("applied effect", slog.String("name", name), slog.String("in", msg), slog.String("out", r))
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

func hteize(msg string) string {
	// We do a slightly higher effort replace-by-word for this effect because
	// a naïve replacer would replace components of words we want to preserve.
	f := strings.Fields(msg)
	for i, w := range f {
		switch w {
		case "":
			// Handle this case so we can check for punctuation more easily later.
			continue
		case "the", "The":
			f[i] = "hte"
		case "THE":
			f[i] = "HTE"
		case "a":
			f[i] = "an"
		case "A":
			// skip this one
		case "an":
			f[i] = "a"
		case "An", "AN":
			f[i] = "A"
		case "your":
			f[i] = "you're"
		case "Your":
			f[i] = "You're"
		case "YOUR":
			f[i] = `YOU"RE`
		case "you're", "you'RE": // tokenization and entropy reduction can lead to you -> ' -> RE
			f[i] = "your"
		case "You're", "You'RE":
			f[i] = "Your"
		case "YOU'RE", "YOU're", `YOU"RE`:
			f[i] = "YORU" // rare treat
		case "there":
			f[i] = thereLower.Pick(rand.Uint32())
		case "There":
			f[i] = thereTitle.Pick(rand.Uint32())
		case "THERE":
			f[i] = thereUpper.Pick(rand.Uint32())
		case "their":
			f[i] = theirLower.Pick(rand.Uint32())
		case "Their":
			f[i] = theirTitle.Pick(rand.Uint32())
		case "THEIR":
			f[i] = theirUpper.Pick(rand.Uint32())
		case "they're":
			f[i] = theyreLower.Pick(rand.Uint32())
		case "They're":
			f[i] = theyreTitle.Pick(rand.Uint32())
		case "THEY'RE", `THEY"RE`:
			f[i] = theyreUpper.Pick(rand.Uint32())
		case "its":
			f[i] = "it's"
		case "Its":
			f[i] = "It's"
		case "ITS":
			f[i] = `IT"S`
		case "it's":
			f[i] = "its"
		case "It's":
			f[i] = "Its"
		case "IT'S", `IT"S`:
			f[i] = "ITS"
		case "then":
			f[i] = "than"
		case "Then":
			f[i] = "Than"
		case "THEN":
			f[i] = "THAN"
		case "than":
			f[i] = "then"
		case "Than":
			f[i] = "Then"
		case "THAN":
			f[i] = "THEN"
		case "should've", "should'VE":
			f[i] = "should of"
		case "Should've", "Should'VE":
			f[i] = "Should of"
		case "SHOULD'VE", "SHOULD've", `SHOULD"VE`:
			f[i] = "SHOULD OF"
		case "it", "It":
			f[i] = "ti"
		case "IT":
			f[i] = "TI"
		default:
			k := strings.IndexAny(w, ".?!")
			switch {
			// Note we check for k > 0, not k >= 0, because we don't want to
			// do this when the punctuation character is at the start.
			case k > 0:
				f[i] = f[i][:k] + " " + f[i][k:]
			case strings.HasSuffix(w, "ing"):
				f[i] = w[:len(w)-2] + "gn"
			case strings.HasSuffix(w, "ING"):
				f[i] = w[:len(w)-2] + "GN"
			}
		}
	}
	return strings.Join(f, " ")
}

var (
	thereLower  = pick.New(pick.FromMap(map[string]int{"their": 2, "they're": 2, "three": 1}))
	thereTitle  = pick.New(pick.FromMap(map[string]int{"Their": 2, "They're": 2, "Three": 1}))
	thereUpper  = pick.New(pick.FromMap(map[string]int{"THEIR": 4, `THEY"RE`: 2, `THEYRE`: 2, "THREE": 1}))
	theirLower  = pick.New(pick.FromMap(map[string]int{"there": 2, "they're": 2}))
	theirTitle  = pick.New(pick.FromMap(map[string]int{"There": 2, "They're": 2}))
	theirUpper  = pick.New(pick.FromMap(map[string]int{"THERE": 4, `THEY"RE`: 2, `THEYRE`: 2}))
	theyreLower = pick.New(pick.FromMap(map[string]int{"there": 2, "their": 2}))
	theyreTitle = pick.New(pick.FromMap(map[string]int{"There": 2, "Their": 2}))
	theyreUpper = pick.New(pick.FromMap(map[string]int{"THERE": 4, "THEIR": 4}))
)
