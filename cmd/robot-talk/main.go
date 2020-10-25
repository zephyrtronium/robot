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
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/zephyrtronium/robot/brain"
)

func main() {
	var source string
	var tag string
	var echo string
	var n int
	var cpu, mem string
	flag.StringVar(&source, "source", "", "database to think from")
	flag.StringVar(&tag, "tag", "", "tag to think from")
	flag.StringVar(&echo, "echo", "", "directory to echo messages to (no echoing if not given)")
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
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		msg := br.Talk(ctx, tag, chain, 1024)
		fmt.Println(msg)
		go doEcho(&wg, msg, tag, echo)
	}
	wg.Wait()
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

func doEcho(wg *sync.WaitGroup, msg, tag, echo string) {
	defer wg.Done()
	if echo == "" {
		return
	}
	f, err := ioutil.TempFile(echo, tag)
	if err != nil {
		log.Println("error opening echo file:", err)
		return
	}
	if _, err := f.WriteString(msg); err != nil {
		log.Println("error writing file:", err)
		return
	}
	if err := f.Close(); err != nil {
		log.Println("error closing file:", err)
		return
	}
}
