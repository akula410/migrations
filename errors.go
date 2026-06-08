package migrations

import "errors"

var (
	ErrMigrationNotFound       = errors.New("migration not found")
	ErrMigrationAlreadyApplied = errors.New("migration already applied")
	ErrMigrationNotApplied     = errors.New("migration not applied")
	ErrChecksumMismatch        = errors.New("migration checksum mismatch")
	ErrLockNotAcquired         = errors.New("migration lock not acquired")
	ErrDirtyState              = errors.New("migration dirty state")
	ErrInvalidIdentifier       = errors.New("invalid identifier")
	ErrDuplicateVersion        = errors.New("duplicate migration version")
)
