package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// mysqlLock implements Lock using MySQL advisory locks (GET_LOCK / RELEASE_LOCK).
//
// MySQL GET_LOCK is connection-scoped: the lock belongs to the connection that called
// GET_LOCK and is automatically released when that connection closes. Acquire therefore
// obtains a dedicated *sql.Conn and keeps it alive until Release is called, which runs
// RELEASE_LOCK on the same connection and then closes it.
type mysqlLock struct {
	db      *sql.DB
	name    string
	timeout time.Duration

	mu       sync.Mutex
	conn     *sql.Conn // non-nil only while lock is held
	acquired bool
}

func newMySQLLock(db *sql.DB, name string, timeout time.Duration) *mysqlLock {
	return &mysqlLock{db: db, name: name, timeout: timeout}
}

// Acquire obtains the MySQL advisory lock on a dedicated connection.
// Returns an error if the lock is already held by this instance.
func (l *mysqlLock) Acquire(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.acquired {
		return fmt.Errorf("lock %q: already acquired", l.name)
	}

	conn, err := l.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("lock: get conn: %w", err)
	}

	secs := int(l.timeout.Seconds())
	if secs < 0 {
		secs = 0
	}

	var result sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", l.name, secs).Scan(&result); err != nil {
		_ = conn.Close()
		return fmt.Errorf("lock: acquire: %w", err)
	}

	// GET_LOCK returns 1 = success, 0 = timeout, NULL = error.
	if !result.Valid || result.Int64 != 1 {
		_ = conn.Close()
		return fmt.Errorf("lock %q: %w", l.name, ErrLockNotAcquired)
	}

	l.conn = conn
	l.acquired = true
	return nil
}

// Release runs RELEASE_LOCK on the same dedicated connection used for Acquire and then
// closes that connection. Returns an error if the lock was not previously acquired.
func (l *mysqlLock) Release(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.acquired || l.conn == nil {
		return fmt.Errorf("lock %q: not acquired", l.name)
	}

	conn := l.conn
	l.conn = nil
	l.acquired = false

	defer conn.Close()

	var result sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", l.name).Scan(&result); err != nil {
		return fmt.Errorf("lock: release: %w", err)
	}
	// RELEASE_LOCK returns 1 = released, 0 = not owned, NULL = not exists.
	if !result.Valid || result.Int64 != 1 {
		return fmt.Errorf("lock: release %q returned %v (not owned or does not exist)", l.name, result)
	}
	return nil
}
