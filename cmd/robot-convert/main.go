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

// robot-convert imports old Robot serialization files to the new brain format.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"strings"

	"github.com/zephyrtronium/robot/brain"
)

func main() {
	var me, tag, in, out string
	flag.StringVar(&me, "me", "", "bot username")
	flag.StringVar(&tag, "tag", "", "input tag")
	flag.StringVar(&in, "in", "", "input json database")
	flag.StringVar(&out, "out", "", "output SQLite3 database")
	flag.Parse()
	if me == "" {
		log.Fatalln("-me is required")
	}
	if tag == "" {
		log.Fatalln("-tag is required")
	}
	if in == "" {
		log.Fatalln("-in is required")
	}
	if out == "" {
		log.Fatalln("-out is required")
	}

	b, err := ioutil.ReadFile(in)
	if err != nil {
		log.Fatalln(err)
	}
	var chain map[string][]string
	if err := json.Unmarshal(b, &chain); err != nil {
		log.Fatalln(err)
	}

	var order int
	for p := range chain {
		order = len(strings.Fields(p)) // easiest consistent measure
		break
	}

	ctx := context.Background()
	br, err := brain.Configure(ctx, out, me, order)
	if err != nil {
		log.Fatalln(err)
	}
	defer br.Close()
	// SQLite: Always use WAL journaling. It can be orders of magnitude faster.
	if _, err := br.Exec(ctx, `PRAGMA journal_mode=WAL`); err != nil {
		log.Println("unable to set journal mode to WAL:", err)
	}
	pre := make([]sql.NullString, order)
	var n int64
	for p, s := range chain {
		f := strings.Fields(p)
		for i, v := range f {
			if v == "\x01" {
				pre[i] = sql.NullString{}
			} else {
				pre[i] = sql.NullString{String: v, Valid: true}
			}
		}
		for _, w := range s {
			switch w {
			case "\x01":
				continue // ??
			case "\x00":
				r, err := br.LearnTuple(ctx, tag, pre, sql.NullString{})
				if err != nil {
					log.Printf("error learning %v -> NULL: %v", pre, err)
					continue
				}
				k, _ := r.RowsAffected()
				n += k
			default:
				r, err := br.LearnTuple(ctx, tag, pre, sql.NullString{String: w, Valid: true})
				if err != nil {
					log.Printf("error learning %v -> %s: %v", pre, w, err)
					continue
				}
				k, _ := r.RowsAffected()
				n += k
			}
		}
	}
	log.Println("learned", n, "chains")
}
