package migrations_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	migrations "github.com/akula410/migrations"
)

func newMockDB(t *testing.T) (interface {
	Applied(context.Context) ([]migrations.AppliedMigration, error)
	Insert(context.Context, migrations.AppliedMigration) error
	Delete(context.Context, string) error
	CreateTable(context.Context) error
}, sqlmock.Sqlmock) {
	t.Helper()
	// We test the mysqlStore behaviour by exercising the full Migrator with a
	// sqlmock db, validating queries at the mock level.
	return nil, nil
}

// storeFromMock creates a *sql.DB backed by sqlmock for direct store testing.
func storeCreateTableTest(t *testing.T) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	m, err := migrations.NewMigrator(db,
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(true),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestStore_CreateTable(t *testing.T) {
	storeCreateTableTest(t)
}

func TestStore_InsertAppliedMigration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mig := newFake("20260101000000", "create_users")

	// Init
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	// Applied query
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum", "applied_at", "execution_time_ms"}))
	// Insert
	mock.ExpectExec(`INSERT INTO`).
		WithArgs(mig.Version(), mig.Name(), migrations.Checksum(mig), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(false),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Up: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestStore_ReadAppliedMigrations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mig := newFake("20260101000000", "create_users")
	at := time.Now().UTC().Truncate(time.Second)

	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows(
			[]string{"version", "name", "checksum", "applied_at", "execution_time_ms"},
		).AddRow(mig.Version(), mig.Name(), migrations.Checksum(mig), at, uint64(123)))

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	applied, err := m.Applied(context.Background())
	if err != nil {
		t.Fatalf("Applied: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(applied))
	}
	if applied[0].Version != mig.Version() {
		t.Fatalf("expected version %s, got %s", mig.Version(), applied[0].Version)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestStore_DeleteOnDown(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mig := newFake("20260101000000", "create_users")
	at := time.Now().UTC()
	cs := migrations.Checksum(mig)

	// Applied query for DownSteps
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows(
			[]string{"version", "name", "checksum", "applied_at", "execution_time_ms"},
		).AddRow(mig.Version(), mig.Name(), cs, at, uint64(0)))
	// Delete
	mock.ExpectExec(`DELETE FROM`).
		WithArgs(mig.Version()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.DownSteps(context.Background(), 1); err != nil {
		t.Fatalf("DownSteps: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestStore_ChecksumMismatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mig := newFake("20260101000000", "create_users")
	at := time.Now().UTC()

	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows(
			[]string{"version", "name", "checksum", "applied_at", "execution_time_ms"},
		).AddRow(mig.Version(), mig.Name(), "badhash", at, uint64(0)))

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = m.Pending(context.Background())
	if err == nil {
		t.Fatal("expected ErrChecksumMismatch")
	}
	if !errors.Is(err, migrations.ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
