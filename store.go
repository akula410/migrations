package migrations

import "context"

// Store manages the migration history table.
type Store interface {
	CreateTable(ctx context.Context) error
	Applied(ctx context.Context) ([]AppliedMigration, error)
	Insert(ctx context.Context, am AppliedMigration) error
	Delete(ctx context.Context, version string) error
	// Dirty returns the first migration with success=0, or nil if none exists.
	Dirty(ctx context.Context) (*DirtyMigration, error)
	// MarkFailed records a failed migration attempt (success=0) using an upsert.
	MarkFailed(ctx context.Context, failed FailedMigration) error
}
