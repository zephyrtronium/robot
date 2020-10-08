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

// robot-talk uses a Robot brain to generate messages.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/zephyrtronium/robot/brain"
)

func main() {
	var source string
	var tag string
	var n int
	var cpu, mem string
	flag.StringVar(&source, "source", "", "database to think from")
	flag.StringVar(&tag, "tag", "", "tag to think from")
	flag.IntVar(&n, "n", 1, "number of times to think")
	flag.StringVar(&cpu, "cpu", "", "CPU profile output file")
	flag.StringVar(&mem, "mem", "", "memory profile output file")
	flag.Parse()

	if cpu != "" {
		f, err := os.Create(cpu)
		if err != nil {
			log.Fatalln("error creating CPU profile output:", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalln("error starting CPU profile:", err)
		}
		defer pprof.StopCPUProfile()
	}
	ctx := context.Background()
	br, err := brain.Open(ctx, source)
	defer br.Close()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := br.Exec(ctx, `PRAGMA wal_checkpoint`); err != nil {
		log.Println("unable to perform WAL checkpoint:", err)
	}
	chain := append([]string{}, flag.Args()...)
	for i := 0; i < n; i++ {
		fmt.Println(br.Talk(ctx, tag, chain, 1024))
	}
	if mem != "" {
		f, err := os.Create(mem)
		if err != nil {
			log.Fatalln("error creating memory profile output:", err)
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Println("error writing heap profile:", err)
		}
	}
}
