package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type mysqlStore struct {
	db        *sql.DB
	tableName string
}

func newMySQLStore(db *sql.DB, tableName string) *mysqlStore {
	return &mysqlStore{db: db, tableName: tableName}
}

// quoteIdent wraps a validated identifier in backticks for MySQL.
func quoteIdent(s string) string {
	return "`" + s + "`"
}

func (s *mysqlStore) CreateTable(ctx context.Context) error {
	tbl := quoteIdent(s.tableName)
	query := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    id                BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    version           VARCHAR(64)      NOT NULL,
    name              VARCHAR(255)     NOT NULL,
    checksum          VARCHAR(128)     NOT NULL,
    direction         ENUM('up','down') NOT NULL DEFAULT 'up',
    success           TINYINT(1)       NOT NULL DEFAULT 1,
    execution_time_ms BIGINT UNSIGNED  NOT NULL DEFAULT 0,
    error_text        TEXT             NULL,
    applied_at        TIMESTAMP        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_version (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`, tbl)

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("store: create table: %w", err)
	}
	return nil
}

func (s *mysqlStore) Applied(ctx context.Context) ([]AppliedMigration, error) {
	tbl := quoteIdent(s.tableName)
	query := fmt.Sprintf(
		`SELECT version, name, checksum, applied_at, execution_time_ms`+
			` FROM %s WHERE direction = 'up' AND success = 1 ORDER BY version ASC`,
		tbl,
	)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("store: query applied: %w", err)
	}
	defer rows.Close()

	var result []AppliedMigration
	for rows.Next() {
		var m AppliedMigration
		var ms uint64
		if err := rows.Scan(&m.Version, &m.Name, &m.Checksum, &m.AppliedAt, &ms); err != nil {
			return nil, fmt.Errorf("store: scan: %w", err)
		}
		m.ExecutionTime = time.Duration(ms) * time.Millisecond
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows: %w", err)
	}
	return result, nil
}

func (s *mysqlStore) Insert(ctx context.Context, am AppliedMigration) error {
	tbl := quoteIdent(s.tableName)
	query := fmt.Sprintf(
		`INSERT INTO %s (version, name, checksum, direction, success, execution_time_ms)`+
			` VALUES (?, ?, ?, 'up', 1, ?)`,
		tbl,
	)
	ms := uint64(am.ExecutionTime / time.Millisecond)
	if _, err := s.db.ExecContext(ctx, query, am.Version, am.Name, am.Checksum, ms); err != nil {
		return fmt.Errorf("store: insert: %w", err)
	}
	return nil
}

// Delete removes the migration record. Called on rollback (down).
// This approach keeps the table in sync with the current schema state.
func (s *mysqlStore) Delete(ctx context.Context, version string) error {
	tbl := quoteIdent(s.tableName)
	query := fmt.Sprintf(`DELETE FROM %s WHERE version = ?`, tbl)
	if _, err := s.db.ExecContext(ctx, query, version); err != nil {
		return fmt.Errorf("store: delete: %w", err)
	}
	return nil
}
