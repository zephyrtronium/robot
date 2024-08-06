package tpool_test

import (
	"sync"
	"testing"

	"github.com/zephyrtronium/robot/tpool"
)

func TestAllocs(t *testing.T) {
	const iters, runs int = 1e3, 1e3
	u := testing.AllocsPerRun(runs, func() {
		var pool sync.Pool
		for range iters {
			x, _ := pool.Get().(*int)
			if x == nil {
				x = new(int)
			}
			pool.Put(x)
			pool.Put(new(int))
		}
	})
	v := testing.AllocsPerRun(runs, func() {
		var pool tpool.Pool[*int]
		for range iters {
			x := pool.Get()
			if x == nil {
				x = new(int)
			}
			pool.Put(x)
			pool.Put(new(int))
		}
	})
	if u != v {
		t.Errorf("different allocs per run: sync.Pool has %v, tpool.Pool[*int] has %v", u, v)
	}
}

func TestAllocsNew(t *testing.T) {
	const iters, runs int = 1e3, 1e3
	u := testing.AllocsPerRun(runs, func() {
		pool := sync.Pool{New: func() any { return new(int) }}
		for range iters {
			x, _ := pool.Get().(*int)
			pool.Put(x)
			pool.Put(new(int))
		}
	})
	v := testing.AllocsPerRun(runs, func() {
		pool := tpool.Pool[*int]{New: func() any { return new(int) }}
		for range iters {
			x := pool.Get()
			pool.Put(x)
			pool.Put(new(int))
		}
	})
	if u != v {
		t.Errorf("different allocs per run: sync.Pool has %v, tpool.Pool[*int] has %v", u, v)
	}
}
