package userhash_test

import (
	"context"
	"testing"
	"time"

	"gitlab.com/zephyrtronium/sq"

	"github.com/zephyrtronium/robot/v2/brain/userhash"

	_ "github.com/mattn/go-sqlite3" // driver
)

// TestHashSQL tests that userhashes round-trip in an SQL database.
func TestHashSQL(t *testing.T) {
	cases := []struct {
		driver string
		dsn    string
		create string
	}{
		{
			driver: "sqlite3",
			dsn:    ":memory:",
			create: `CREATE TABLE test (x BLOB) STRICT`,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.driver, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db, err := sq.Open(c.driver, c.dsn)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Ping(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(ctx, c.create); err != nil {
				t.Fatal(err)
			}
			in := &userhash.Hash{}
			for i := range in {
				in[i] = 0x55
			}
			r, err := db.Exec(ctx, `INSERT INTO test (x) VALUES (?)`, in)
			if err != nil {
				t.Errorf("couldn't insert userhash: %v", err)
			}
			n, _ := r.RowsAffected()
			if n != 1 {
				t.Errorf("wrong number of rows affected: want 1, got %d", n)
			}
			var out userhash.Hash
			if err := db.QueryRow(ctx, `SELECT x FROM test`).Scan(&out); err != nil {
				t.Errorf("couldn't scan userhash: %v", err)
			}
			if out != *in {
				t.Errorf("wrong scan result: want %x, got %x", *in, out)
			}
		})
	}
}

func TestHasher(t *testing.T) {
	t.Parallel()
	// Every combination of key, time, user, and location must produce a
	// distinct userhash.
	keys := []string{
		"madoka",
		"homura",
	}
	users := []string{
		"bocchi",
		"nijika",
		"ryou",
		"kita",
	}
	locs := []string{
		"kaguya",
		"miyuki",
	}
	times := []time.Time{
		time.Unix(0, -userhash.TimeQuantum.Nanoseconds()),
		time.Unix(0, 0),
		time.Unix(0, userhash.TimeQuantum.Nanoseconds()),
	}
	u := make(map[userhash.Hash]bool, len(keys)*len(users)*len(locs)*len(times))
	for _, key := range keys {
		for _, user := range users {
			for _, loc := range locs {
				for _, when := range times {
					hr := userhash.New([]byte(key))
					a := *hr.Hash(new(userhash.Hash), user, loc, when)
					if u[a] {
						t.Errorf("duplicate hash: %s/%s/%s/%v gave %x", key, user, loc, when, a)
					}
					u[a] = true
					b := *hr.Hash(new(userhash.Hash), user, loc, when)
					if a != b {
						t.Errorf("repeated hash changed: %s/%s/%s/%v gave first %x then %x", key, user, loc, when, a, b)
					}
				}
			}
		}
	}
}
