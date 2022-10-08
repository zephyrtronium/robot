package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/zephyrtronium/robot/v2/brain/sqlbrain"
	"gitlab.com/zephyrtronium/sq"

	_ "github.com/mattn/go-sqlite3" // driver
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	var (
		driver  string
		connect string
		order   int
	)
	flag.StringVar(&driver, "driver", "sqlite3", "database engine")
	flag.StringVar(&connect, "connect", "", "connection string")
	flag.IntVar(&order, "order", 0, "prefix length to configure")
	flag.Parse()
	if order <= 0 {
		log.Fatal("order is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		// Make the second ^C always kill the program by restoring default
		// interrupt behavior after the first.
		<-ctx.Done()
		cancel()
	}()

	db, err := sq.Open(driver, connect)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	if err := sqlbrain.Create(ctx, db, order); err != nil {
		log.Fatal(err)
	}
}
