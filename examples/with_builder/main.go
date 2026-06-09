//go:build ignore

// Example: using github.com/akula410/builder/v2 to construct migration SQL.
//
// Requirements:
//
//	go get github.com/akula410/builder/v2
//
// Run:
//
//	MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/db?parseTime=true' go run examples/with_builder/main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	sqlbuilder "github.com/akula410/builder/v2"
	migrations "github.com/akula410/migrations/v2"
)

// CreateTagsTable20260608000003 builds its schema using the builder DDL API.
type CreateTagsTable20260608000003 struct{}

func (m CreateTagsTable20260608000003) Version() string { return "20260608000003" }
func (m CreateTagsTable20260608000003) Name() string    { return "create_tags_table" }

func (m CreateTagsTable20260608000003) Up(ctx context.Context, db *sql.DB) error {
	// Use the builder DDL API to construct the CREATE TABLE statement safely.
	exec := sqlbuilder.NewExecutor(db)

	_, err := exec.ExecContext(ctx,
		sqlbuilder.CreateTable("tags").
			IfNotExists().
			Column(sqlbuilder.Col("id").BigIntUnsigned().NotNull().AutoIncrement()).
			Column(sqlbuilder.Col("name").VarChar(100).NotNull()).
			Column(sqlbuilder.Col("created_at").Timestamp().NotNull().Default("CURRENT_TIMESTAMP")).
			PrimaryKey("id").
			UniqueIndex("uq_tags_name", "name").
			Engine("InnoDB").
			Collate("utf8mb4_unicode_ci"),
	)
	return err
}

func (m CreateTagsTable20260608000003) Down(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS tags`)
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
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(
			CreateTagsTable20260608000003{},
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
