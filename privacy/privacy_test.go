package privacy_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/zephyrtronium/robot/privacy"
)

var dbcount atomic.Uint64

func testConn() *sqlitex.Pool {
	k := dbcount.Add(1)
	pool, err := sqlitex.NewPool(fmt.Sprintf("file:%d.db?mode=memory&cache=shared", k), sqlitex.PoolOptions{Flags: sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenMemory | sqlite.OpenSharedCache | sqlite.OpenURI})
	if err != nil {
		panic(err)
	}
	return pool
}

func TestList(t *testing.T) {
	type check struct {
		user string
		ok   bool
	}
	cases := []struct {
		name string
		add  []string
		rem  []string
		chk  []check
	}{
		{
			name: "empty",
			add:  nil,
			rem:  nil,
			chk: []check{
				{user: "bocchi", ok: false},
				{user: "ryou", ok: false},
				{user: "nijika", ok: false},
				{user: "kita", ok: false},
			},
		},
		{
			name: "present",
			add: []string{
				"bocchi",
				"ryou",
			},
			rem: nil,
			chk: []check{
				{user: "bocchi", ok: true},
				{user: "ryou", ok: true},
				{user: "nijika", ok: false},
				{user: "kita", ok: false},
			},
		},
		{
			name: "remove-none",
			add: []string{
				"bocchi",
				"ryou",
			},
			rem: []string{
				"nijika",
				"kita",
			},
			chk: []check{
				{user: "bocchi", ok: true},
				{user: "ryou", ok: true},
				{user: "nijika", ok: false},
				{user: "kita", ok: false},
			},
		},
		{
			name: "remove",
			add: []string{
				"bocchi",
				"ryou",
				"nijika",
				"kita",
			},
			rem: []string{
				"nijika",
				"kita",
			},
			chk: []check{
				{user: "bocchi", ok: true},
				{user: "ryou", ok: true},
				{user: "nijika", ok: false},
				{user: "kita", ok: false},
			},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := testConn()
			l, err := privacy.Open(ctx, db)
			if err != nil {
				t.Fatal(err)
			}
			for _, v := range c.add {
				if err := l.Add(ctx, v); err != nil {
					t.Errorf("couldn't add %q: %v", v, err)
				}
			}
			for _, v := range c.rem {
				if err := l.Remove(ctx, v); err != nil {
					t.Errorf("couldn't remove %q: %v", v, err)
				}
			}
			for _, v := range c.chk {
				err := l.Check(ctx, v.user)
				switch err {
				case nil:
					if v.ok {
						t.Errorf("%q not in list but should be", v.user)
					}
				case privacy.ErrPrivate:
					if !v.ok {
						t.Errorf("%q in list but shouldn't be", v.user)
					}
				default:
					t.Errorf("couldn't check for %q in list: %v", v.user, err)
				}
			}
		})
	}
}
