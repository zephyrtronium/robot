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

// robot-init initializes or reconfigures a Robot brain.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"time"

	"github.com/zephyrtronium/robot/brain"
)

func main() {
	var source, conf string
	flag.StringVar(&source, "source", "", "SQL database source")
	flag.StringVar(&conf, "conf", "", "config JSON file")
	flag.Parse()

	b, err := ioutil.ReadFile(conf)
	if err != nil {
		log.Fatal(err)
	}
	var config config
	if err := json.Unmarshal(b, &config); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	var br *brain.Brain
	if config.Me == "" && config.Prefix == 0 {
		// Update rather than create.
		log.Println("opening", source, "with existing name/order")
		br, err = brain.Open(ctx, source)
	} else {
		// Create, or at least reconfigure.
		log.Println("opening", source, "with name", config.Me, "and order", config.Prefix)
		br, err = brain.Configure(ctx, source, config.Me, config.Prefix)
	}
	if err != nil {
		log.Fatal(err)
	}

	if config.Block != "" {
		log.Println("setting default block expression to", config.Block)
		if _, err := br.Exec(ctx, `UPDATE config SET block=? WHERE id=1`, config.Block); err != nil {
			log.Fatal(err)
		}
	}
	for channel, cfg := range config.Chans {
		uchan(ctx, br, channel, cfg)
	}
	if config.Emotes != nil {
		log.Println("configuring global emotes")
		gemotes(ctx, br, config.Emotes)
	}
	if config.Privs != nil {
		log.Println("configuring global privileges")
		gprivs(ctx, br, config.Privs)
	}

	br.Close()
	log.Println("all done")
}

func uchan(ctx context.Context, br *brain.Brain, channel string, cfg channel) {
	log.Println("configuring", channel)
	if _, err := br.Exec(ctx, `INSERT OR IGNORE INTO chans(name) VALUES (?)`, channel); err != nil {
		log.Printf("ERROR inserting %s: %v (SKIP)", channel, err)
		return
	}
	if cfg.Learn != nil {
		learn := sql.NullString{String: *cfg.Learn, Valid: *cfg.Learn != ""}
		_, err := br.Exec(ctx, `UPDATE chans SET learn=? WHERE name=?`, learn, channel)
		if err != nil {
			log.Printf("ERROR setting learn tag on %s: %v", channel, err)
		}
	}
	if cfg.Send != nil {
		send := sql.NullString{String: *cfg.Send, Valid: *cfg.Send != ""}
		_, err := br.Exec(ctx, `UPDATE chans SET send=? WHERE name=?`, send, channel)
		if err != nil {
			log.Printf("ERROR setting send tag on %s: %v", channel, err)
		}
	}
	if cfg.Lim != 0 {
		_, err := br.Exec(ctx, `UPDATE chans SET lim=? WHERE name=?`, cfg.Lim, channel)
		if err != nil {
			log.Printf("ERROR setting lim on %s: %v", channel, err)
		}
	}
	if cfg.Prob != nil {
		_, err := br.Exec(ctx, `UPDATE chans SET prob=? WHERE name=?`, *cfg.Prob, channel)
		if err != nil {
			log.Printf("ERROR setting prob on %s: %v", channel, err)
		}
	}
	if cfg.Rate != 0 {
		_, err := br.Exec(ctx, `UPDATE chans SET rate=? WHERE name=?`, cfg.Rate, channel)
		if err != nil {
			log.Printf("ERROR setting rate on %s: %v", channel, err)
		}
	}
	if cfg.Burst != 0 {
		_, err := br.Exec(ctx, `UPDATE chans SET burst=? WHERE name=?`, cfg.Burst, channel)
		if err != nil {
			log.Printf("ERROR setting burst on %s: %v", channel, err)
		}
	}
	if cfg.Block != "" {
		_, err := br.Exec(ctx, `UPDATE chans SET block=? WHERE name=?`, cfg.Block, channel)
		if err != nil {
			log.Printf("ERROR setting block on %s: %v", channel, err)
		}
	}
	if cfg.Respond != nil {
		_, err := br.Exec(ctx, `UPDATE chans SET respond=? WHERE name=?`, *cfg.Respond, channel)
		if err != nil {
			log.Printf("ERROR setting respond on %s: %v", channel, err)
		}
	}
	if cfg.Silence != nil {
		_, err := br.Exec(ctx, `UPDATE chans SET silence=? WHERE name=?`, *cfg.Silence, channel)
		if err != nil {
			log.Printf("ERROR setting silence on %s: %v", channel, err)
		}
	}
	if cfg.Emotes != nil {
		log.Println("configuring", channel, "emotes")
		uemotes(ctx, br, channel, cfg)
	}
	if cfg.Privs != nil {
		log.Println("configuring", channel, "privileges")
		uprivs(ctx, br, channel, cfg)
	}
	br.Exec(ctx, `PRAGMA wal_checkpoint;`) // ignore error
}

func uemotes(ctx context.Context, br *brain.Brain, channel string, cfg channel) {
	row := br.QueryRow(ctx, `SELECT send FROM chans WHERE name=?`, channel)
	var tag sql.NullString
	if err := row.Scan(&tag); err != nil {
		log.Printf("ERROR scanning send tag for %s: %v", channel, err)
		return
	}
	if !tag.Valid {
		log.Println("can't update emotes without a send tag")
		return
	}
	if cfg.Emotes != nil {
		_, err := br.Exec(ctx, `DELETE FROM emotes WHERE tag=?`, tag.String)
		if err != nil {
			log.Printf("ERROR removing existing emotes for %s: %v", channel, err)
		}
		for _, em := range cfg.Emotes {
			_, err := br.Exec(ctx, `INSERT INTO emotes(tag, emote, weight) VALUES (?, ?, ?)`, tag, em.E, em.W)
			if err != nil {
				log.Printf("ERROR inserting emote %q: %v", em.E, err)
			}
		}
	}
}

func uprivs(ctx context.Context, br *brain.Brain, channel string, cfg channel) {
	if _, err := br.Exec(ctx, `DELETE FROM privs WHERE chan=?`, channel); err != nil {
		log.Printf("ERROR removing existing privs for %s: %v", channel, err)
		return
	}
	for _, priv := range cfg.Privs {
		_, err := br.Exec(ctx, `INSERT INTO privs(user, chan, priv) VALUES (?, ?, ?)`, priv.User, channel, priv.Priv)
		if err != nil {
			log.Printf("ERROR setting privilege for %s: %v", priv.User, err)
		}
	}
}

func gemotes(ctx context.Context, br *brain.Brain, ems []emote) {
	if _, err := br.Exec(ctx, `DELETE FROM emotes WHERE tag IS NULL`); err != nil {
		log.Println("ERROR removing existing global emotes:", err)
		return
	}
	for _, em := range ems {
		_, err := br.Exec(ctx, `INSERT INTO emotes(emote, weight) VALUES (?, ?)`, em.E, em.W)
		if err != nil {
			log.Printf("ERROR inserting emote %q: %v", em.E, err)
		}
	}
}

func gprivs(ctx context.Context, br *brain.Brain, privs []privilege) {
	if _, err := br.Exec(ctx, `DELETE FROM privs WHERE chan IS NULL`); err != nil {
		log.Println("ERROR removing existing global privs:", err)
		return
	}
	for _, priv := range privs {
		_, err := br.Exec(ctx, `INSERT INTO privs(user, priv) VALUES (?, ?)`, priv.User, priv.Priv)
		if err != nil {
			log.Printf("ERROR setting global privilege for %s: %v", priv.User, err)
		}
	}
}

type config struct {
	Me     string             `json:"me"`
	Prefix int                `json:"prefix"`
	Block  string             `json:"block"`
	Chans  map[string]channel `json:"chans"`
	Emotes []emote            `json:"emotes"`
	Privs  []privilege        `json:"privs"`
}

type channel struct {
	Learn   *string     `json:"learn"`
	Send    *string     `json:"send"`
	Lim     int         `json:"lim"`
	Prob    *float64    `json:"prob"`
	Rate    float64     `json:"rate"`
	Burst   int         `json:"burst"`
	Block   string      `json:"block"`
	Respond *bool       `json:"respond"`
	Silence *time.Time  `json:"silence"`
	Emotes  []emote     `json:"emotes"`
	Privs   []privilege `json:"privs"`
}

type emote struct {
	E string `json:"e"`
	W int    `json:"w"`
}

type privilege struct {
	User string `json:"user"`
	Priv string `json:"priv"`
}
