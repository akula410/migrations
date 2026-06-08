//go:build ignore

// Example: connect (connection) + builder (SQL DSL) + migrations.
//
// Requirements:
//
//	go get github.com/akula410/connect/v2
//	go get github.com/akula410/builder/v2
//
// Run:
//
//	MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/db?parseTime=true' go run examples/with_connect_and_builder/main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	sqlbuilder "github.com/akula410/builder/v2"
	connect "github.com/akula410/connect/v2"

	migrations "github.com/akula410/migrations"
)

// CreateCommentsTable20260608000004 uses builder for DDL and connect for the connection.
type CreateCommentsTable20260608000004 struct{}

func (m CreateCommentsTable20260608000004) Version() string { return "20260608000004" }
func (m CreateCommentsTable20260608000004) Name() string    { return "create_comments_table" }

func (m CreateCommentsTable20260608000004) Up(ctx context.Context, db *sql.DB) error {
	exec := sqlbuilder.NewExecutor(db)
	_, err := exec.ExecContext(ctx,
		sqlbuilder.CreateTable("comments").
			IfNotExists().
			Column(sqlbuilder.Col("id").BigIntUnsigned().NotNull().AutoIncrement()).
			Column(sqlbuilder.Col("post_id").BigIntUnsigned().NotNull()).
			Column(sqlbuilder.Col("body").Text().NotNull()).
			Column(sqlbuilder.Col("created_at").Timestamp().NotNull().Default("CURRENT_TIMESTAMP")).
			PrimaryKey("id").
			Engine("InnoDB").
			Collate("utf8mb4_unicode_ci"),
	)
	return err
}

func (m CreateCommentsTable20260608000004) Down(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS comments`)
	return err
}

func cfgFromDSN() connect.Config {
	dsn := os.Getenv("MYSQL_DSN")
	cfg := connect.Config{User: "root", DBName: "test_db"}
	if dsn == "" {
		return cfg
	}
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
	return cfg
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Connect via akula410/connect
	db, err := connect.NewMySQLContext(ctx, cfgFromDSN())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer connect.Close(db)

	// 2. Create migrator with builder-based migrations
	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(
			CreateCommentsTable20260608000004{},
		),
		migrations.WithLogger(log.Default()),
	)
	if err != nil {
		log.Fatalf("NewMigrator: %v", err)
	}

	// 3. Init + Up
	if err := m.Init(ctx); err != nil {
		log.Fatalf("Init: %v", err)
	}
	if err := m.Up(ctx); err != nil {
		log.Fatalf("Up: %v", err)
	}

	// 4. Status
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

	// 5. Optional: roll back the last migration
	if err := m.DownSteps(ctx, 1); err != nil {
		log.Fatalf("DownSteps: %v", err)
	}
	fmt.Println("rolled back successfully")
}
