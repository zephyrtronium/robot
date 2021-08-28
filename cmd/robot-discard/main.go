/*
Copyright (C) 2021  Branden J Brown

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

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/zephyrtronium/robot/brain"
)

const query = `
WITH candidates AS (
	SELECT id FROM tuples%[1]d WHERE tag=?1
	ORDER BY id
	LIMIT (SELECT CAST(count(*)*?2 AS INTEGER) FROM tuples%[1]d WHERE tag=?1)
	)
DELETE FROM tuples%[1]d WHERE id IN candidates;
`

func main() {
	var source, tags string
	var p float64
	flag.StringVar(&source, "source", "", "SQL database source")
	flag.StringVar(&tags, "tags", "", "space-separated tags, all learn tags if empty")
	flag.Float64Var(&p, "p", 0, "proportion of chains to discard in [0, 1)")
	flag.Parse()

	ctx := context.Background()
	br, err := brain.Open(ctx, source)
	if err != nil {
		log.Fatal(err)
	}
	defer br.Close()
	q := fmt.Sprintf(query, br.Order())

	list, err := taglist(ctx, br, tags)
	if err != nil {
		log.Fatal(err)
	}
	tx, err := br.Tx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		log.Fatal("couldn't open transaction:", err)
	}
	for _, tag := range list {
		r, err := tx.ExecContext(ctx, q, tag, p)
		if err != nil {
			log.Printf("couldn't discard from tag %s: %v", tag, err)
			continue
		}
		n, err := r.RowsAffected()
		if err != nil {
			log.Printf("couldn't get number of rows affected for tag %s: %v", tag, err)
		}
		log.Printf("discarded %d rows with tag %s", n, tag)
	}

	fmt.Println("enter y to commit, anything else to roll back")
	var e string
	fmt.Scanln(&e)
	if strings.TrimSpace(e) == "y" {
		if err := tx.Commit(); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := tx.Rollback(); err != nil {
			log.Fatal(err)
		}
	}
}

func taglist(ctx context.Context, br *brain.Brain, list string) ([]string, error) {
	if list != "" {
		return strings.Fields(list), nil
	}
	rows, err := br.Query(ctx, `SELECT learn FROM chans`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var r []string
	for rows.Next() {
		var s sql.NullString
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("couldn't scan tag: %w", err)
		}
		if !s.Valid {
			continue
		}
		r = append(r, s.String)
	}
	return r, nil
}
