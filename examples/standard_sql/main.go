//go:build ignore

// Example: standard database/sql
//
// Run:
//
//	MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/db?parseTime=true' go run examples/standard_sql/main.go
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
)

// CreateUsersTable20260608000001 is a sample migration.
type CreateUsersTable20260608000001 struct{}

func (m CreateUsersTable20260608000001) Version() string { return "20260608000001" }
func (m CreateUsersTable20260608000001) Name() string    { return "create_users_table" }

func (m CreateUsersTable20260608000001) Up(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			name       VARCHAR(255)    NOT NULL,
			email      VARCHAR(255)    NOT NULL,
			created_at TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uq_users_email (email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	return err
}

func (m CreateUsersTable20260608000001) Down(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS users`)
	return err
}

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
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(
			CreateUsersTable20260608000001{},
		),
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
