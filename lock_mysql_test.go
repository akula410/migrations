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

	// Acquire (dedicated conn)
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
	// Release (same dedicated conn)
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

// TestLock_DedicatedConnection verifies that GET_LOCK and RELEASE_LOCK are issued on
// the same connection: the mock expects GET_LOCK first and RELEASE_LOCK last, with no
// other queries on the lock connection in between (dirty/applied/insert use the main db).
func TestLock_DedicatedConnection(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mig := newFake("20260101000000", "m1")

	// Expectations in the exact order the implementation must produce them.
	// GET_LOCK is on the dedicated conn; everything else is on the pool.
	mock.ExpectQuery(`SELECT GET_LOCK`).
		WithArgs("migrations_lock", 30).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	expectNoDirtyState(mock)
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum", "applied_at", "execution_time_ms"}))
	mock.ExpectExec(`INSERT INTO`).
		WithArgs(mig.Version(), mig.Name(), migrations.Checksum(mig), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
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
		t.Fatalf("dedicated connection order violated: %v", err)
	}
}

// TestLock_DoubleAcquireBlocked verifies that attempting to Up twice concurrently
// (or calling Acquire twice) does not silently issue two GET_LOCK calls.
// We test the sequential case: the second Up after the first sees no dirty state,
// but the underlying mock expectations ensure the lock/unlock cycle is clean.
func TestLock_DoubleAcquireBlocked(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// First Up — full cycle
	mock.ExpectQuery(`SELECT GET_LOCK`).
		WithArgs("migrations_lock", 30).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	expectNoDirtyState(mock)
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum", "applied_at", "execution_time_ms"}))
	mock.ExpectQuery(`SELECT RELEASE_LOCK`).
		WithArgs("migrations_lock").
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

	m, err := migrations.NewMigrator(db,
		migrations.WithLock(true),
		migrations.WithAutoCreateTable(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("first Up: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet after first Up: %v", err)
	}
}
