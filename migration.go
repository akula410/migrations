package migrations

import (
	"context"
	"database/sql"
	"time"
)

// Migration is the core interface every migration must implement.
type Migration interface {
	Version() string
	Name() string
	Up(ctx context.Context, db *sql.DB) error
	Down(ctx context.Context, db *sql.DB) error
}

// TxMigration is an optional interface for migrations that must run inside a transaction.
// The runner provides a *sql.Tx. Implement Migration in addition to TxMigration when you
// also want to support TransactionNone mode for the same migration.
type TxMigration interface {
	Version() string
	Name() string
	UpTx(ctx context.Context, tx *sql.Tx) error
	DownTx(ctx context.Context, tx *sql.Tx) error
}

// ChecksumMigration is an optional interface for migrations that provide their own checksum.
// Implement this to detect SQL body changes; the default checksum only covers Version+Name.
// If Checksum() returns an empty string, the default sha256(version+name) is used as fallback.
type ChecksumMigration interface {
	Checksum() string
}

// StatusItem describes the current state of a single migration.
type StatusItem struct {
	Version       string
	Name          string
	Applied       bool
	Dirty         bool   // true when the last attempt failed (success=0 in the store)
	DirtyError    string // error message from the failed attempt
	AppliedAt     time.Time
	Checksum      string
	ExecutionTime time.Duration
}

// AppliedMigration is a migration that has been recorded in the history store.
type AppliedMigration struct {
	Version       string
	Name          string
	Checksum      string
	AppliedAt     time.Time
	ExecutionTime time.Duration
}

// DirtyMigration is a migration whose last execution failed (success=0 in the store).
type DirtyMigration struct {
	Version       string
	Name          string
	Checksum      string
	Direction     string // "up" or "down"
	ErrorText     string
	AppliedAt     time.Time
	ExecutionTime time.Duration
}

// FailedMigration carries the data to be written when a migration fails.
type FailedMigration struct {
	Version       string
	Name          string
	Checksum      string
	Direction     string // "up" or "down"
	ErrorText     string
	ExecutionTime time.Duration
}
