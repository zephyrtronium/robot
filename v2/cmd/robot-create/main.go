package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"strings"
	"text/template"

	"github.com/zephyrtronium/robot/v2/brain/sqlbrain"

	_ "github.com/mattn/go-sqlite3" // driver
)

//go:embed *.sql
var templateFiles embed.FS
var templates = template.Must(template.ParseFS(templateFiles, "*.sql"))

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

	data := struct {
		Driver  string
		N, NM1  int
		Version int
		Iter    []struct{}
	}{
		Driver: driver,
		N:      order, NM1: order - 1,
		Version: sqlbrain.SchemaVersion,
		Iter:    make([]struct{}, order),
	}
	var query strings.Builder
	files, err := fs.ReadDir(templateFiles, ".")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		err := templates.ExecuteTemplate(&query, file.Name(), &data)
		if err != nil {
			log.Fatalf("couldn't interpret %s: %v", file.Name(), err)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		// Make the second ^C always kill the program by restoring default
		// interrupt behavior after the first.
		<-ctx.Done()
		cancel()
	}()

	db, err := sql.Open(driver, connect)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, query.String()); err != nil {
		log.Fatalf("couldn't exec: %v", err)
	}
	if err := tx.Commit(); err != nil {
		log.Fatalf("couldn't commit: %v", err)
	}
}
