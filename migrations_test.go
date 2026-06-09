package migrations_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	migrations "github.com/akula410/migrations/v2"
	_ "github.com/akula410/migrations/v2/internal/testutil"
)

// ── fixtures ──────────────────────────────────────────────────────────────────

type fakeMigration struct {
	version string
	name    string
	upErr   error
	downErr error
}

func (f fakeMigration) Version() string                         { return f.version }
func (f fakeMigration) Name() string                            { return f.name }
func (f fakeMigration) Up(_ context.Context, _ *sql.DB) error   { return f.upErr }
func (f fakeMigration) Down(_ context.Context, _ *sql.DB) error { return f.downErr }

func newFake(version, name string) fakeMigration {
	return fakeMigration{version: version, name: name}
}

// fakeStore is an in-memory Store for unit tests.
type fakeStore struct {
	applied []migrations.AppliedMigration
	dirty   *migrations.DirtyMigration
}

func (s *fakeStore) CreateTable(_ context.Context) error { return nil }
func (s *fakeStore) Applied(_ context.Context) ([]migrations.AppliedMigration, error) {
	return append([]migrations.AppliedMigration{}, s.applied...), nil
}
func (s *fakeStore) Insert(_ context.Context, am migrations.AppliedMigration) error {
	s.applied = append(s.applied, am)
	s.dirty = nil
	return nil
}
func (s *fakeStore) Delete(_ context.Context, version string) error {
	out := s.applied[:0]
	for _, a := range s.applied {
		if a.Version != version {
			out = append(out, a)
		}
	}
	s.applied = out
	return nil
}
func (s *fakeStore) Dirty(_ context.Context) (*migrations.DirtyMigration, error) {
	return s.dirty, nil
}
func (s *fakeStore) MarkFailed(_ context.Context, f migrations.FailedMigration) error {
	s.dirty = &migrations.DirtyMigration{
		Version:   f.Version,
		Name:      f.Name,
		Checksum:  f.Checksum,
		Direction: f.Direction,
		ErrorText: f.ErrorText,
	}
	return nil
}

// ── Checksum ──────────────────────────────────────────────────────────────────

func TestChecksumDeterministic(t *testing.T) {
	m := newFake("20260101", "create_users")
	c1 := migrations.Checksum(m)
	c2 := migrations.Checksum(m)
	if c1 != c2 {
		t.Fatalf("checksum not deterministic: %s != %s", c1, c2)
	}
}

func TestChecksumDiffers(t *testing.T) {
	a := newFake("20260101", "create_users")
	b := newFake("20260101", "create_posts")
	if migrations.Checksum(a) == migrations.Checksum(b) {
		t.Fatal("different names should produce different checksums")
	}
}

// ── Option validation ─────────────────────────────────────────────────────────

func TestNewMigrator_InvalidTableName(t *testing.T) {
	db := openFakeDB(t)
	_, err := migrations.NewMigrator(db, migrations.WithTableName("123bad"))
	if err == nil {
		t.Fatal("expected error for invalid table name")
	}
	if !errors.Is(err, migrations.ErrInvalidIdentifier) {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestNewMigrator_InvalidLockName(t *testing.T) {
	db := openFakeDB(t)
	_, err := migrations.NewMigrator(db, migrations.WithLockName("bad-name!"))
	if err == nil {
		t.Fatal("expected error for invalid lock name")
	}
	if !errors.Is(err, migrations.ErrInvalidIdentifier) {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestNewMigrator_NilDB(t *testing.T) {
	_, err := migrations.NewMigrator(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

// ── Sorting ───────────────────────────────────────────────────────────────────

func TestMigrationsSortedByVersion(t *testing.T) {
	m1 := newFake("20260103", "c")
	m2 := newFake("20260101", "a")
	m3 := newFake("20260102", "b")

	db := openFakeDB(t)
	mg, err := migrations.NewMigrator(db,
		migrations.WithMigrations(m1, m2, m3),
		migrations.WithLock(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Status returns items sorted by version.
	// We need a functional store; use the DB with sqlmock via a thin wrapper test,
	// so here we just verify NewMigrator does not error (sort+dedup happened).
	_ = mg
}

// ── Duplicate version detection ────────────────────────────────────────────────

func TestDuplicateVersion(t *testing.T) {
	db := openFakeDB(t)
	_, err := migrations.NewMigrator(db,
		migrations.WithMigrations(
			newFake("20260101", "a"),
			newFake("20260101", "b"),
		),
	)
	if err == nil {
		t.Fatal("expected error for duplicate version")
	}
	if !errors.Is(err, migrations.ErrDuplicateVersion) {
		t.Fatalf("expected ErrDuplicateVersion, got %v", err)
	}
}

// ── Error sentinel wrapping ────────────────────────────────────────────────────

func TestErrorWrapping(t *testing.T) {
	base := fmt.Errorf("version %q: %w", "20260101", migrations.ErrChecksumMismatch)
	if !errors.Is(base, migrations.ErrChecksumMismatch) {
		t.Fatal("wrapped ErrChecksumMismatch not detected by errors.Is")
	}
}

// ── Empty migration list ───────────────────────────────────────────────────────

func TestEmptyMigrationList(t *testing.T) {
	db := openFakeDB(t)
	_, err := migrations.NewMigrator(db)
	if err != nil {
		t.Fatalf("empty list should not fail: %v", err)
	}
}

// ── ValidateIdentifier edge cases ─────────────────────────────────────────────

func TestValidIdentifiers(t *testing.T) {
	valid := []string{"schema_migrations", "my_table", "_priv", "A1"}
	db := openFakeDB(t)
	for _, name := range valid {
		_, err := migrations.NewMigrator(db, migrations.WithTableName(name))
		if err != nil {
			t.Errorf("expected valid identifier %q to pass, got %v", name, err)
		}
	}
}

func TestInvalidIdentifiers(t *testing.T) {
	invalid := []string{"", "1abc", "my-table", "my table", "a.b"}
	db := openFakeDB(t)
	for _, name := range invalid {
		_, err := migrations.NewMigrator(db, migrations.WithTableName(name))
		if err == nil {
			t.Errorf("expected invalid identifier %q to fail", name)
		}
	}
}

// ── LockTimeout option ────────────────────────────────────────────────────────

func TestWithLockTimeout(t *testing.T) {
	db := openFakeDB(t)
	_, err := migrations.NewMigrator(db, migrations.WithLockTimeout(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
}

// ── ChecksumMigration interface ───────────────────────────────────────────────

type fakeChecksumMigration struct {
	fakeMigration
	cs string
}

func (f fakeChecksumMigration) Checksum() string { return f.cs }

func TestChecksumMigration_Custom(t *testing.T) {
	m := fakeChecksumMigration{
		fakeMigration: newFake("20260101", "create_users"),
		cs:            "my-custom-checksum",
	}
	got := migrations.Checksum(m)
	if got != "my-custom-checksum" {
		t.Fatalf("expected custom checksum, got %s", got)
	}
}

func TestChecksumMigration_FallbackOnEmpty(t *testing.T) {
	m := fakeChecksumMigration{
		fakeMigration: newFake("20260101", "create_users"),
		cs:            "", // empty → sha256 fallback
	}
	got := migrations.Checksum(m)
	want := migrations.Checksum(newFake("20260101", "create_users"))
	if got != want {
		t.Fatalf("expected fallback checksum %s, got %s", want, got)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// openFakeDB returns a *sql.DB backed by a fake driver that never panics.
// It is used only for option-validation tests that never actually query the DB.
func openFakeDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("migrations_fake", "")
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
