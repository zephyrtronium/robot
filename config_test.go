package main_test

import (
	"context"
	_ "embed"
	"strings"
	"testing"

	main "github.com/zephyrtronium/robot"
)

//go:embed example.toml
var exampleToml string

func eqcase[T comparable](t *testing.T, name string, val T, eq T) {
	t.Helper()
	if val != eq {
		t.Errorf("wrong %s: want %#v, got %#v", name, eq, val)
	}
}

func TestExampleConfig(t *testing.T) {
	cfg, _, err := main.Load(context.Background(), strings.NewReader(exampleToml))
	if err != nil {
		t.Errorf("failed to load example.toml: %v", err)
	}

	eqcase(t, "Owner.Name", cfg.Owner.Name, `zephyrtronium`)
	eqcase(t, "Owner.Contact", cfg.Owner.Contact, `/w zephyrtronium`)
	eqcase(t, "DB.KVBrain", cfg.DB.KVBrain, "")
	eqcase(t, "DB.KVFlag", cfg.DB.KVFlag, "")
	eqcase(t, "HTTP.Listen", cfg.HTTP.Listen, ":4959")
	eqcase(t, "Global.Block", cfg.Global.Block, `(?i)bad\s+stuff[^$x]`)
	eqcase(t, "Global.Emotes[``]", cfg.Global.Emotes[``], 4)
	eqcase(t, "Global.Emotes[`;)`]", cfg.Global.Emotes[`;)`], 1)
	eqcase(t, "Global.Effects[``]", cfg.Global.Effects[``], 18)
	eqcase(t, "Global.Effects[`OwO`]", cfg.Global.Effects[`OwO`], 1)
	eqcase(t, "Global.Effects[`AAAAA`]", cfg.Global.Effects[`AAAAA`], 0)
	eqcase(t, "Global.Effects[`o`]", cfg.Global.Effects[`o`], 1)
	eqcase(t, "Global.Privileges.Twitch[0].Name", cfg.Global.Privileges.Twitch[0].Name, "nightbot")
	eqcase(t, "Global.Privileges.Twitch[0].Level", cfg.Global.Privileges.Twitch[0].Level, "ignore")
	eqcase(t, "TMI.CID", cfg.TMI.CID, `hof5gwx0su6owfnys0nyan9c87zr6t`)
	eqcase(t, "TMI.RedirectURL", cfg.TMI.RedirectURL, `http://localhost`)
	eqcase(t, "TMI.TokenFile", cfg.TMI.TokenFile, `/var/robot/tmi_refresh`)
	eqcase(t, "TMI.Owner.ID", cfg.TMI.Owner.ID, `51421897`)
	eqcase(t, "TMI.Owner.Name", cfg.TMI.Owner.Name, `zephyrtronium`)
	eqcase(t, "TMI.Rate.Every", cfg.TMI.Rate.Every, 30)
	eqcase(t, "TMI.Rate.Num", cfg.TMI.Rate.Num, 20)
	eqcase(t, "Twitch[`bocchi`].Channels[0]", cfg.Twitch[`bocchi`].Channels[0], `#bocchi`)
	eqcase(t, "Twitch[`bocchi`].Learn", cfg.Twitch[`bocchi`].Learn, `bocchi`)
	eqcase(t, "Twitch[`bocchi`].Send", cfg.Twitch[`bocchi`].Send, `bocchi`)
	eqcase(t, "Twitch[`bocchi`].Block", cfg.Twitch[`bocchi`].Block, `(?i)cucumber[^$x]`)
	eqcase(t, "Twitch[`bocchi`].Responses", cfg.Twitch[`bocchi`].Responses, 0.02)
	eqcase(t, "Twitch[`bocchi`].Rate.Every", cfg.Twitch[`bocchi`].Rate.Every, 10.1)
	eqcase(t, "Twitch[`bocchi`].Rate.Num", cfg.Twitch[`bocchi`].Rate.Num, 2)
	eqcase(t, "Twitch[`bocchi`].Copypasta.Need", cfg.Twitch[`bocchi`].Copypasta.Need, 2)
	eqcase(t, "Twitch[`bocchi`].Copypasta.Within", cfg.Twitch[`bocchi`].Copypasta.Within, 30)
	eqcase(t, "Twitch[`bocchi`].Privileges[0].Name", cfg.Twitch[`bocchi`].Privileges[0].Name, `zephyrtronium`)
	eqcase(t, "Twitch[`bocchi`].Privileges[0].Level", cfg.Twitch[`bocchi`].Privileges[0].Level, `moderator`)
	eqcase(t, "Twitch[`bocchi`].Emotes[`btw`]", cfg.Twitch[`bocchi`].Emotes[`btw make sure to stretch, hydrate, and take care of yourself <3`], 1)
	eqcase(t, "Twitch[`bocchi`].Effects[`AAAAA`]", cfg.Twitch[`bocchi`].Effects[`AAAAA`], 44444)
	substrings := []struct {
		name string
		val  string
		has  string
	}{
		{"SecretFile", cfg.SecretFile, "/key"},
		{"DB.SQLBrain", cfg.DB.SQLBrain, "file:"},
		{"DB.Privacy", cfg.DB.Privacy, "file:"},
		{"DB.Spoken", cfg.DB.Spoken, "file:"},
		{"TMI.SecretFile", cfg.TMI.SecretFile, "/twitch_client_secret"},
	}
	for _, c := range substrings {
		if !strings.Contains(c.val, c.has) {
			t.Errorf("wrong %s: %q does not contain %q", c.name, c.val, c.has)
		}
	}
}
