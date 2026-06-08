//go:build ignore

// Example: using auto-generated migrations from the migrations/ sub-package.
//
// Run:
//
//	MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/db?parseTime=true' go run examples/generated_migrations/main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	migrations "github.com/akula410/migrations"
	mymigrations "github.com/akula410/migrations/examples/generated_migrations/migrations"
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("MYSQL_DSN is required")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mymigrations.List...),
		migrations.WithLogger(log.Default()),
	)
	if err != nil {
		log.Fatalf("NewMigrator: %v", err)
	}

	if err := m.Init(ctx); err != nil {
		log.Fatalf("Init: %v", err)
	}
	if err := m.Up(ctx); err != nil {
		log.Fatalf("Up: %v", err)
	}

	status, err := m.Status(ctx)
	if err != nil {
		log.Fatalf("Status: %v", err)
	}

	fmt.Printf("\n%-16s  %-25s  %-8s\n", "VERSION", "NAME", "STATUS")
	for _, s := range status {
		st := "pending"
		if s.Applied {
			st = "applied"
		}
		fmt.Printf("%-16s  %-25s  %-8s\n", s.Version, s.Name, st)
	}
}
