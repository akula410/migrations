package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	migrations "github.com/akula410/migrations/v2"
)

func dsn(t *testing.T) string {
	t.Helper()
	v := os.Getenv("MYSQL_DSN")
	if v == "" {
		t.Skip("MYSQL_DSN not set; skipping integration tests")
	}
	return v
}

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", dsn(t))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type intMig struct {
	version string
	name    string
	upSQL   string
	downSQL string
}

func (m intMig) Version() string { return m.version }
func (m intMig) Name() string    { return m.name }
func (m intMig) Up(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, m.upSQL)
	return err
}
func (m intMig) Down(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, m.downSQL)
	return err
}

func TestIntegration_FullLifecycle(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()

	tableName := "int_test_schema_migrations"

	// Cleanup before and after.
	_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `int_test_users`")
	_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `"+tableName+"`")
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `int_test_users`")
		_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `"+tableName+"`")
	})

	m1 := intMig{
		version: "20260101000001",
		name:    "create_int_test_users",
		upSQL: `CREATE TABLE int_test_users (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		downSQL: `DROP TABLE IF EXISTS int_test_users`,
	}

	mg, err := migrations.NewMigrator(db,
		migrations.WithMigrations(m1),
		migrations.WithTableName(tableName),
	)
	if err != nil {
		t.Fatalf("NewMigrator: %v", err)
	}

	// Init creates the schema table.
	if err := mg.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Up applies the migration.
	if err := mg.Up(ctx); err != nil {
		t.Fatalf("Up: %v", err)
	}

	// Status shows it as applied.
	status, err := mg.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(status) != 1 || !status[0].Applied {
		t.Fatalf("expected 1 applied migration, got %+v", status)
	}

	// Validate passes.
	if err := mg.Validate(ctx); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// Up again is idempotent (no pending).
	if err := mg.Up(ctx); err != nil {
		t.Fatalf("second Up: %v", err)
	}

	// Down rolls back.
	if err := mg.DownSteps(ctx, 1); err != nil {
		t.Fatalf("Down: %v", err)
	}

	applied, err := mg.Applied(ctx)
	if err != nil {
		t.Fatalf("Applied: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("expected 0 applied after Down, got %d", len(applied))
	}
}

func TestIntegration_ChecksumMismatch(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()

	tableName := "int_test_checksum_migrations"

	_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `int_test_checksum_users`")
	_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `"+tableName+"`")
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `int_test_checksum_users`")
		_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `"+tableName+"`")
	})

	m1 := intMig{
		version: "20260102000001",
		name:    "create_int_test_checksum_users",
		upSQL:   `CREATE TABLE int_test_checksum_users (id INT PRIMARY KEY) ENGINE=InnoDB`,
		downSQL: `DROP TABLE IF EXISTS int_test_checksum_users`,
	}

	mg, err := migrations.NewMigrator(db,
		migrations.WithMigrations(m1),
		migrations.WithTableName(tableName),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := mg.Init(ctx); err != nil {
		t.Fatal(err)
	}
	if err := mg.Up(ctx); err != nil {
		t.Fatal(err)
	}

	// Tamper: create a migrator with a different name for the same version.
	m1tampered := intMig{version: m1.version, name: "tampered_name"}
	mg2, err := migrations.NewMigrator(db,
		migrations.WithMigrations(m1tampered),
		migrations.WithTableName(tableName),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mg2.Pending(ctx)
	if !errors.Is(err, migrations.ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
}

func TestIntegration_UpSteps(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()

	tableName := "int_test_steps_migrations"
	_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `int_steps_t1`")
	_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `int_steps_t2`")
	_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `"+tableName+"`")
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `int_steps_t1`")
		_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `int_steps_t2`")
		_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS `"+tableName+"`")
	})

	m1 := intMig{version: "20260103000001", name: "t1", upSQL: "CREATE TABLE int_steps_t1 (id INT PRIMARY KEY) ENGINE=InnoDB", downSQL: "DROP TABLE IF EXISTS int_steps_t1"}
	m2 := intMig{version: "20260103000002", name: "t2", upSQL: "CREATE TABLE int_steps_t2 (id INT PRIMARY KEY) ENGINE=InnoDB", downSQL: "DROP TABLE IF EXISTS int_steps_t2"}

	mg, err := migrations.NewMigrator(db,
		migrations.WithMigrations(m1, m2),
		migrations.WithTableName(tableName),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := mg.Init(ctx); err != nil {
		t.Fatal(err)
	}

	// Apply only 1
	if err := mg.UpSteps(ctx, 1); err != nil {
		t.Fatalf("UpSteps(1): %v", err)
	}
	applied, _ := mg.Applied(ctx)
	if len(applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(applied))
	}

	// Apply remaining
	if err := mg.Up(ctx); err != nil {
		t.Fatalf("Up remaining: %v", err)
	}
	applied, _ = mg.Applied(ctx)
	if len(applied) != 2 {
		t.Fatalf("expected 2 applied, got %d", len(applied))
	}

	// Down 1 step
	if err := mg.DownSteps(ctx, 1); err != nil {
		t.Fatalf("DownSteps(1): %v", err)
	}
	applied, _ = mg.Applied(ctx)
	if len(applied) != 1 {
		t.Fatalf("expected 1 applied after rollback, got %d", len(applied))
	}
}
