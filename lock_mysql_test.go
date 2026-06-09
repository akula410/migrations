package migrations_test

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	migrations "github.com/akula410/migrations/v2"
)

func TestLock_AcquireSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mig := newFake("20260101000000", "m1")

	// Acquire
	mock.ExpectQuery(`SELECT GET_LOCK`).
		WithArgs("migrations_lock", 30).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	// Dirty check
	expectNoDirtyState(mock)
	// Applied (pending check — no applied migrations, so mig is pending)
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum", "applied_at", "execution_time_ms"}))
	// Insert (UPSERT)
	mock.ExpectExec(`INSERT INTO`).
		WithArgs(mig.Version(), mig.Name(), migrations.Checksum(mig), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Release
	mock.ExpectQuery(`SELECT RELEASE_LOCK`).
		WithArgs("migrations_lock").
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(true),
		migrations.WithAutoCreateTable(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Up: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLock_AcquireTimeout(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// GET_LOCK returns 0 → timeout
	mock.ExpectQuery(`SELECT GET_LOCK`).
		WithArgs("migrations_lock", 30).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(0))

	m, err := migrations.NewMigrator(db,
		migrations.WithLock(true),
		migrations.WithAutoCreateTable(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	err = m.Up(context.Background())
	if err == nil {
		t.Fatal("expected ErrLockNotAcquired")
	}
	if !errors.Is(err, migrations.ErrLockNotAcquired) {
		t.Fatalf("expected ErrLockNotAcquired, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLock_ReleaseLockError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Acquire succeeds
	mock.ExpectQuery(`SELECT GET_LOCK`).
		WithArgs("migrations_lock", 30).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	// Dirty check
	expectNoDirtyState(mock)
	// Applied: no pending
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum", "applied_at", "execution_time_ms"}))
	// Release returns 0 (not owned) — should not fail Up itself
	mock.ExpectQuery(`SELECT RELEASE_LOCK`).
		WithArgs("migrations_lock").
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(0))

	m, err := migrations.NewMigrator(db,
		migrations.WithLock(true),
		migrations.WithAutoCreateTable(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Up itself should succeed even if release returns non-1
	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Up should not fail on release error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLock_Disabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// No GET_LOCK/RELEASE_LOCK expected
	// Dirty check
	expectNoDirtyState(mock)
	// Applied: no pending
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum", "applied_at", "execution_time_ms"}))

	m, err := migrations.NewMigrator(db,
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Up: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
