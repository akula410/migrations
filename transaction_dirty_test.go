package migrations_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	migrations "github.com/akula410/migrations/v2"
)

// fakeBothMigration implements both Migration and TxMigration.
type fakeBothMigration struct {
	version   string
	name      string
	upErr     error
	downErr   error
	upTxErr   error
	downTxErr error
}

func (f fakeBothMigration) Version() string                           { return f.version }
func (f fakeBothMigration) Name() string                              { return f.name }
func (f fakeBothMigration) Up(_ context.Context, _ *sql.DB) error     { return f.upErr }
func (f fakeBothMigration) Down(_ context.Context, _ *sql.DB) error   { return f.downErr }
func (f fakeBothMigration) UpTx(_ context.Context, _ *sql.Tx) error   { return f.upTxErr }
func (f fakeBothMigration) DownTx(_ context.Context, _ *sql.Tx) error { return f.downTxErr }

func newFakeBoth(version, name string) fakeBothMigration {
	return fakeBothMigration{version: version, name: name}
}

// ── TransactionNone ────────────────────────────────────────────────────────────

// TestTransactionNone_NoBeginTx verifies that TransactionNone never creates a
// database transaction, even when the migration implements TxMigration.
func TestTransactionNone_NoBeginTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// fakeBothMigration implements both Migration and TxMigration.
	// In TransactionPerMigration mode the runner would call BeginTx → UpTx → Commit.
	// In TransactionNone mode it must call Up(db) with NO BeginTx.
	mig := newFakeBoth("20260101000000", "m1")

	// Dirty check
	expectNoDirtyState(mock)
	// Applied: no previously applied migrations
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum", "applied_at", "execution_time_ms"}))
	// Insert after successful Up — no ExpectBegin/Commit registered
	mock.ExpectExec(`INSERT INTO`).
		WithArgs(mig.Version(), mig.Name(), migrations.Checksum(mig), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(false),
		migrations.WithTransactionMode(migrations.TransactionNone),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Up with TransactionNone: %v", err)
	}
	// If sqlmock detects an unexpected BeginTx call it will surface here.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations (unexpected BeginTx?): %v", err)
	}
}

// ── TransactionAll ─────────────────────────────────────────────────────────────

// TestTransactionAll_ErrorForNonTxMigration verifies that TransactionAll rejects
// a plain Migration that does not implement TxMigration.
func TestTransactionAll_ErrorForNonTxMigration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mig := newFake("20260101000000", "m1") // implements Migration only

	// Dirty check
	expectNoDirtyState(mock)
	// Applied: migration is pending
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum", "applied_at", "execution_time_ms"}))
	// No BeginTx expected — error is returned before entering the tx path.

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(false),
		migrations.WithTransactionMode(migrations.TransactionAll),
	)
	if err != nil {
		t.Fatal(err)
	}

	gotErr := m.Up(context.Background())
	if gotErr == nil {
		t.Fatal("expected error for non-TxMigration with TransactionAll")
	}
	if !strings.Contains(gotErr.Error(), "transaction all requires TxMigration") {
		t.Fatalf("unexpected error message: %v", gotErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestTransactionAll_SuccessForTxMigration verifies that TransactionAll wraps
// all TxMigration migrations in a single shared transaction.
func TestTransactionAll_SuccessForTxMigration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mig := newFakeBoth("20260101000000", "m1")

	// Dirty check
	expectNoDirtyState(mock)
	// Applied: no applied migrations
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum", "applied_at", "execution_time_ms"}))
	// Shared transaction
	mock.ExpectBegin()
	mock.ExpectCommit()
	// Insert after commit
	mock.ExpectExec(`INSERT INTO`).
		WithArgs(mig.Version(), mig.Name(), migrations.Checksum(mig), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(false),
		migrations.WithTransactionMode(migrations.TransactionAll),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Up with TransactionAll: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ── Dirty state ────────────────────────────────────────────────────────────────

// TestDirtyState_BlocksUp verifies that a dirty row (success=0) prevents Up.
func TestDirtyState_BlocksUp(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mig := newFake("20260101000000", "m1")

	// Dirty check returns a failed row
	mock.ExpectQuery(`COALESCE`).
		WillReturnRows(sqlmock.NewRows(
			[]string{"version", "name", "checksum", "direction", "error_text", "applied_at", "execution_time_ms"},
		).AddRow(mig.Version(), mig.Name(), "somehash", "up", "table already exists", time.Now(), uint64(0)))

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	gotErr := m.Up(context.Background())
	if gotErr == nil {
		t.Fatal("expected ErrDirtyState")
	}
	if !errors.Is(gotErr, migrations.ErrDirtyState) {
		t.Fatalf("expected ErrDirtyState, got %v", gotErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestDirtyState_WithAllowDirty verifies that WithAllowDirty skips the dirty check.
func TestDirtyState_WithAllowDirty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mig := newFake("20260101000000", "m1")

	// No COALESCE (dirty) query — WithAllowDirty skips it.
	// Applied: no applied migrations
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum", "applied_at", "execution_time_ms"}))
	// Insert
	mock.ExpectExec(`INSERT INTO`).
		WithArgs(mig.Version(), mig.Name(), migrations.Checksum(mig), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(false),
		migrations.WithAllowDirty(true),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Up with AllowDirty: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestDirtyState_MarkFailedOnUpError verifies that a failed Up records a dirty row.
func TestDirtyState_MarkFailedOnUpError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	upErr := fmt.Errorf("table already exists")
	mig := fakeMigration{version: "20260101000000", name: "m1", upErr: upErr}

	// Dirty check: clean
	expectNoDirtyState(mock)
	// Applied: no applied migrations
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum", "applied_at", "execution_time_ms"}))
	// MarkFailed INSERT (version, name, checksum, direction="up", ms, error_text)
	mock.ExpectExec(`INSERT INTO`).
		WithArgs(mig.Version(), mig.Name(), migrations.Checksum(mig), "up", sqlmock.AnyArg(), upErr.Error()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	gotErr := m.Up(context.Background())
	if gotErr == nil {
		t.Fatal("expected up error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations (MarkFailed not called?): %v", err)
	}
}

// TestDirtyState_MarkFailedOnDownError verifies that a failed Down records a dirty row.
func TestDirtyState_MarkFailedOnDownError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	downErr := fmt.Errorf("cannot drop table")
	mig := fakeMigration{version: "20260101000000", name: "m1", downErr: downErr}
	at := time.Now().UTC()
	cs := migrations.Checksum(mig)

	// Dirty check: clean
	expectNoDirtyState(mock)
	// Applied: migration is applied
	mock.ExpectQuery(`SELECT version`).
		WillReturnRows(sqlmock.NewRows(
			[]string{"version", "name", "checksum", "applied_at", "execution_time_ms"},
		).AddRow(mig.Version(), mig.Name(), cs, at, uint64(0)))
	// MarkFailed INSERT (version, name, checksum, direction="down", ms, error_text)
	mock.ExpectExec(`INSERT INTO`).
		WithArgs(mig.Version(), mig.Name(), cs, "down", sqlmock.AnyArg(), downErr.Error()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	m, err := migrations.NewMigrator(db,
		migrations.WithMigrations(mig),
		migrations.WithLock(false),
		migrations.WithAutoCreateTable(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	gotErr := m.DownSteps(context.Background(), 1)
	if gotErr == nil {
		t.Fatal("expected down error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations (MarkFailed not called?): %v", err)
	}
}
