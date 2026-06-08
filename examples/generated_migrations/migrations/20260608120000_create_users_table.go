package migrations

import (
	"context"
	"database/sql"
)

type CreateUsersTable20260608120000 struct{}

func (m CreateUsersTable20260608120000) Version() string { return "20260608120000" }
func (m CreateUsersTable20260608120000) Name() string    { return "create_users_table" }

func (m CreateUsersTable20260608120000) Up(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			name       VARCHAR(255)    NOT NULL,
			created_at TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	return err
}

func (m CreateUsersTable20260608120000) Down(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS users`)
	return err
}
