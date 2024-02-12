package main

import "testing"

func TestParseCommand(t *testing.T) {
	cases := []struct {
		name string
		me   string
		in   string
		text string
		ok   bool
	}{
		{"empty", "Bocchi", "", "", false},
		{"exact", "Bocchi", "Bocchi", "", true},
		{"case", "Bocchi", "bOCCHI", "", true},
		{"prespace", "Bocchi", " Bocchi", "", true},
		{"postspace", "Bocchi", "Bocchi ", "", true},
		{"at", "Bocchi", "@Bocchi", "", true},
		{"punct", "Bocchi", "Bocchi...", "", true},
		{"prefix", "Bocchi", "Bocchi3", "", false},
		{"suffix", "Bocchi", "9Bocchi", "", false},
		{"text-after", "Bocchi", "Bocchi the Rock!", "the Rock!", true},
		{"text-before", "Bocchi", "Hitori Bocchi", "Hitori", true},
		{"middle", "Bocchi", "Hitori Bocchi Tokyo", "", false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got, ok := parseCommand(c.me, c.in)
			if got != c.text {
				t.Errorf("wrong command text: want %q, got %q", c.text, got)
			}
			if ok != c.ok {
				t.Errorf("wrong commandness: want %t, got %t", c.ok, ok)
			}
		})
	}
}
