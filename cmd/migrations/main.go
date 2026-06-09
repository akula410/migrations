// Command migrations is a CLI example for github.com/akula410/migrations/v2.
//
// Usage:
//
//	export MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/db?parseTime=true'
//	migrations init
//	migrations create create_users_table
//	migrations up
//	migrations up --steps=1
//	migrations down --steps=1
//	migrations status
//	migrations validate
//
// This binary does not ship any migrations of its own — it is intended as a
// starting point that you copy and extend with your own migration list.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	migrations "github.com/akula410/migrations/v2"
)

// Register your migrations here.
var migrationList []migrations.Migration

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "create":
		runCreate()
	case "init", "up", "down", "status", "validate":
		runDB(cmd)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: migrations <command> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  create <name>       Generate a new migration file")
	fmt.Fprintln(os.Stderr, "  init                Create the schema_migrations table")
	fmt.Fprintln(os.Stderr, "  up [--steps N]      Apply pending migrations")
	fmt.Fprintln(os.Stderr, "  down [--steps N]    Roll back applied migrations")
	fmt.Fprintln(os.Stderr, "  status              Print migration status")
	fmt.Fprintln(os.Stderr, "  validate            Validate applied checksums")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Environment:")
	fmt.Fprintln(os.Stderr, "  MYSQL_DSN           MySQL DSN (required for db commands)")
	fmt.Fprintln(os.Stderr, "  MIGRATIONS_DIR      Output dir for 'create' (default: migrations)")
}

func runCreate() {
	if len(os.Args) < 3 {
		log.Fatal("usage: migrations create <name>")
	}
	dir := envOr("MIGRATIONS_DIR", "migrations")
	if err := migrations.GenerateMigration(migrations.GenerateOptions{
		Dir:  dir,
		Name: os.Args[2],
	}); err != nil {
		log.Fatalf("create: %v", err)
	}
	fmt.Printf("created migration in %s\n", dir)
}

func runDB(cmd string) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("MYSQL_DSN is required")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(migrationList...),
		migrations.WithLogger(log.Default()),
	)
	if err != nil {
		log.Fatalf("migrator: %v", err)
	}

	switch cmd {
	case "init":
		if err := m.Init(ctx); err != nil {
			log.Fatalf("init: %v", err)
		}
		fmt.Println("schema_migrations table ready")

	case "up":
		fs := flag.NewFlagSet("up", flag.ExitOnError)
		steps := fs.Int("steps", 0, "number of migrations to apply (0 = all)")
		_ = fs.Parse(os.Args[2:])
		if err := m.UpSteps(ctx, *steps); err != nil {
			log.Fatalf("up: %v", err)
		}

	case "down":
		fs := flag.NewFlagSet("down", flag.ExitOnError)
		steps := fs.Int("steps", 1, "number of migrations to roll back")
		_ = fs.Parse(os.Args[2:])
		if err := m.DownSteps(ctx, *steps); err != nil {
			log.Fatalf("down: %v", err)
		}

	case "status":
		items, err := m.Status(ctx)
		if err != nil {
			log.Fatalf("status: %v", err)
		}
		printStatus(items)

	case "validate":
		if err := m.Validate(ctx); err != nil {
			log.Fatalf("validate: %v", err)
		}
		fmt.Println("all applied migrations are valid")
	}
}

func printStatus(items []migrations.StatusItem) {
	fmt.Printf("%-16s  %-32s  %-8s  %s\n", "VERSION", "NAME", "STATUS", "APPLIED AT")
	fmt.Println(strings.Repeat("-", 80))
	for _, it := range items {
		status := "pending"
		at := ""
		if it.Dirty {
			status = "dirty"
		} else if it.Applied {
			status = "applied"
			at = it.AppliedAt.Format(time.RFC3339)
		}
		fmt.Printf("%-16s  %-32s  %-8s  %s\n", it.Version, it.Name, status, at)
		if it.Dirty && it.DirtyError != "" {
			fmt.Printf("  error: %s\n", it.DirtyError)
		}
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
