package migrations

import "context"

// Store manages the migration history table.
type Store interface {
	CreateTable(ctx context.Context) error
	Applied(ctx context.Context) ([]AppliedMigration, error)
	Insert(ctx context.Context, am AppliedMigration) error
	Delete(ctx context.Context, version string) error
}
