package userhash_test

import (
	"testing"
	"time"

	"github.com/zephyrtronium/robot/userhash"
)

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
