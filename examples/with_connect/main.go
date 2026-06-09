//go:build ignore

// Example: using github.com/akula410/connect/v2 for the database connection.
//
// Requirements:
//
//	go get github.com/akula410/connect/v2
//
// Run:
//
//	MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/db?parseTime=true' go run examples/with_connect/main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	connect "github.com/akula410/connect/v2"

	migrations "github.com/akula410/migrations/v2"
)

type AddPostsTable20260608000002 struct{}

func (m AddPostsTable20260608000002) Version() string { return "20260608000002" }
func (m AddPostsTable20260608000002) Name() string    { return "add_posts_table" }

func (m AddPostsTable20260608000002) Up(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS posts (
			id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			user_id    BIGINT UNSIGNED NOT NULL,
			title      VARCHAR(255)    NOT NULL,
			created_at TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	return err
}

func (m AddPostsTable20260608000002) Down(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS posts`)
	return err
}

// parseDSN extracts connect.Config fields from a DSN string for demo purposes.
// In production use connect.Config fields directly.
func parseDSN() connect.Config {
	dsn := os.Getenv("MYSQL_DSN")
	// Very simplified parser for demo — assumes user:pass@tcp(host:port)/dbname
	// In practice, set Config fields directly.
	cfg := connect.Config{
		User:   "root",
		DBName: "test_db",
	}
	if dsn != "" {
		if idx := strings.Index(dsn, ":"); idx > 0 {
			cfg.User = dsn[:idx]
			rest := dsn[idx+1:]
			if idx2 := strings.Index(rest, "@"); idx2 > 0 {
				cfg.Password = rest[:idx2]
				rest = rest[idx2+1:]
			}
			if strings.HasPrefix(rest, "tcp(") {
				rest = rest[4:]
				if idx3 := strings.Index(rest, ")"); idx3 > 0 {
					addr := rest[:idx3]
					rest = rest[idx3+1:]
					parts := strings.SplitN(addr, ":", 2)
					cfg.Host = parts[0]
					if len(parts) == 2 {
						cfg.Port = parts[1]
					}
				}
				rest = strings.TrimPrefix(rest, "/")
				if idx4 := strings.Index(rest, "?"); idx4 > 0 {
					cfg.DBName = rest[:idx4]
				} else {
					cfg.DBName = rest
				}
			}
		}
	}
	return cfg
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := parseDSN()

	// connect.NewMySQLContext handles pool settings and PingContext automatically.
	db, err := connect.NewMySQLContext(ctx, cfg)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer connect.Close(db)

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(
			AddPostsTable20260608000002{},
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
