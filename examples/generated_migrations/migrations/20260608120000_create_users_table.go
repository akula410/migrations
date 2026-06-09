package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
)

// createUsersTable20260608120000UpSQL and createUsersTable20260608120000DownSQL are
// package-level constants so that Checksum() automatically detects SQL body changes.
const createUsersTable20260608120000UpSQL = `
	CREATE TABLE IF NOT EXISTS users (
		id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		name       VARCHAR(255)    NOT NULL,
		created_at TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`

const createUsersTable20260608120000DownSQL = `
	DROP TABLE IF EXISTS users
`

type CreateUsersTable20260608120000 struct{}

func (m CreateUsersTable20260608120000) Version() string { return "20260608120000" }
func (m CreateUsersTable20260608120000) Name() string    { return "create_users_table" }

func (m CreateUsersTable20260608120000) Up(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, createUsersTable20260608120000UpSQL)
	return err
}

func (m CreateUsersTable20260608120000) Down(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, createUsersTable20260608120000DownSQL)
	return err
}

// Checksum implements ChecksumMigration. Edit the SQL constants above and the next
// migration run will detect the change via ErrChecksumMismatch.
func (m CreateUsersTable20260608120000) Checksum() string {
	h := sha256.New()
	h.Write([]byte(createUsersTable20260608120000UpSQL))
	h.Write([]byte{0})
	h.Write([]byte(createUsersTable20260608120000DownSQL))
	return fmt.Sprintf("%x", h.Sum(nil))
}
