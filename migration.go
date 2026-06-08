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
// The runner provides a *sql.Tx for each migration when this interface is implemented.
type TxMigration interface {
	Version() string
	Name() string
	UpTx(ctx context.Context, tx *sql.Tx) error
	DownTx(ctx context.Context, tx *sql.Tx) error
}

// StatusItem describes the current state of a single migration.
type StatusItem struct {
	Version       string
	Name          string
	Applied       bool
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
