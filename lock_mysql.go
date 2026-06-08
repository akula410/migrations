package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type mysqlLock struct {
	db      *sql.DB
	name    string
	timeout time.Duration
}

func newMySQLLock(db *sql.DB, name string, timeout time.Duration) *mysqlLock {
	return &mysqlLock{db: db, name: name, timeout: timeout}
}

func (l *mysqlLock) Acquire(ctx context.Context) error {
	secs := int(l.timeout.Seconds())
	if secs < 0 {
		secs = 0
	}

	var result sql.NullInt64
	if err := l.db.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", l.name, secs).Scan(&result); err != nil {
		return fmt.Errorf("lock: acquire: %w", err)
	}

	// GET_LOCK returns 1 = success, 0 = timeout, NULL = error.
	if !result.Valid || result.Int64 != 1 {
		return fmt.Errorf("lock %q: %w", l.name, ErrLockNotAcquired)
	}
	return nil
}

func (l *mysqlLock) Release(ctx context.Context) error {
	var result sql.NullInt64
	if err := l.db.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", l.name).Scan(&result); err != nil {
		return fmt.Errorf("lock: release: %w", err)
	}
	// RELEASE_LOCK returns 1 = released, 0 = not owned, NULL = not exists.
	// We treat non-1 as a warning but not a hard error here.
	if !result.Valid || result.Int64 != 1 {
		return fmt.Errorf("lock: release %q returned %v (not owned or does not exist)", l.name, result)
	}
	return nil
}
