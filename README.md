# migrations

Production-ready MySQL 8 migration runner for Go.

## Features

- `database/sql` as the only required dependency
- MySQL `GET_LOCK` / `RELEASE_LOCK` distributed locking
- Three transaction modes: per-migration, all-in-one, none
- SHA-256 checksum validation of applied migrations
- Migration generator with timestamp-based versioning
- CLI skeleton for `init`, `create`, `up`, `down`, `status`, `validate`
- Optional integration with `github.com/akula410/connect/v2` and `github.com/akula410/builder/v2`

## Compatibility

Only **MySQL 8** is supported. Passing any other dialect to `WithDialect` returns `ErrUnsupportedDialect`.

PostgreSQL, SQLite, and MariaDB are not supported. The package is intentionally MySQL-specific so it can rely on MySQL semantics (advisory locks, `ENUM`, `TINYINT`, `AUTO_INCREMENT`, `ON DUPLICATE KEY UPDATE`).

## Installation

```bash
go get github.com/akula410/migrations/v2
```

## Quick start

```go
package main

import (
    "context"
    "database/sql"
    "log"

    _ "github.com/go-sql-driver/mysql"
    migrations "github.com/akula410/migrations/v2"
)

type CreateUsers20260608000001 struct{}

func (m CreateUsers20260608000001) Version() string { return "20260608000001" }
func (m CreateUsers20260608000001) Name() string    { return "create_users" }

func (m CreateUsers20260608000001) Up(ctx context.Context, db *sql.DB) error {
    _, err := db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS users (
            id   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
            name VARCHAR(255) NOT NULL,
            PRIMARY KEY (id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
    `)
    return err
}

func (m CreateUsers20260608000001) Down(ctx context.Context, db *sql.DB) error {
    _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS users`)
    return err
}

func main() {
    db, _ := sql.Open("mysql", "user:pass@tcp(127.0.0.1:3306)/mydb?parseTime=true")
    defer db.Close()

    m, err := migrations.NewMigrator(db,
        migrations.WithMigrations(CreateUsers20260608000001{}),
        migrations.WithLogger(log.Default()),
    )
    if err != nil { log.Fatal(err) }

    ctx := context.Background()
    if err := m.Init(ctx); err != nil { log.Fatal(err) }
    if err := m.Up(ctx);   err != nil { log.Fatal(err) }
}
```

## Migration interface

```go
type Migration interface {
    Version() string
    Name() string
    Up(ctx context.Context, db *sql.DB) error
    Down(ctx context.Context, db *sql.DB) error
}
```

For migrations that must run inside a transaction, implement `TxMigration`:

```go
type TxMigration interface {
    Version() string
    Name() string
    UpTx(ctx context.Context, tx *sql.Tx) error
    DownTx(ctx context.Context, tx *sql.Tx) error
}
```

## API

```go
func NewMigrator(db *sql.DB, opts ...Option) (*Migrator, error)
func (m *Migrator) Init(ctx context.Context) error
func (m *Migrator) Up(ctx context.Context) error
func (m *Migrator) UpSteps(ctx context.Context, steps int) error
func (m *Migrator) Down(ctx context.Context) error
func (m *Migrator) DownSteps(ctx context.Context, steps int) error
func (m *Migrator) Status(ctx context.Context) ([]StatusItem, error)
func (m *Migrator) Pending(ctx context.Context) ([]Migration, error)
func (m *Migrator) Applied(ctx context.Context) ([]AppliedMigration, error)
func (m *Migrator) Validate(ctx context.Context) error
```

## Options

| Option                             | Default                   | Description                                                |
|------------------------------------|---------------------------|------------------------------------------------------------|
| `WithMigrations(ms ...Migration)`  | —                         | Register migrations. Returns `ErrInvalidMigration` for nil/empty `Version`/`Name`. |
| `WithTableName(name)`              | `schema_migrations`       | History table name                                         |
| `WithDialect(dialect)`             | `MySQL`                   | Only `MySQL` is accepted; others return `ErrUnsupportedDialect`. |
| `WithLogger(logger)`               | noop                      | Progress logger                                            |
| `WithLock(enabled)`                | `true`                    | Enable MySQL `GET_LOCK` on a dedicated connection          |
| `WithLockName(name)`               | `migrations_lock`         | Lock identifier                                            |
| `WithLockTimeout(d)`               | `30s`                     | `GET_LOCK` timeout                                         |
| `WithAutoCreateTable(enabled)`     | `true`                    | Auto-create schema table                                   |
| `WithTransactionMode(mode)`        | `TransactionPerMigration` | Transaction strategy                                       |
| `WithChecksumValidation(enabled)`  | `true`                    | Detect modified migrations                                 |
| `WithAllowDirty(enabled)`          | `false`                   | Skip dirty-state check                                     |

## Example: database/sql

See `examples/standard_sql/main.go`.

## Example: with connect

```go
import connect "github.com/akula410/connect/v2"

db, err := connect.NewMySQLContext(ctx, connect.Config{
    User:     "user",
    Password: "pass",
    Host:     "127.0.0.1",
    Port:     "3306",
    DBName:   "mydb",
})
// pass db to NewMigrator as usual
```

See `examples/with_connect/main.go`.

## Example: with builder

The `builder/v2` column API uses `Column(name, "TYPE")` with plain SQL type strings.
There are no `.VarChar()`, `.BigInt()`, or similar method-based type helpers.

```go
import sqlbuilder "github.com/akula410/builder/v2"

func (m MyMigration) Up(ctx context.Context, db *sql.DB) error {
    exec := sqlbuilder.NewExecutor(db)
    _, err := exec.ExecContext(ctx,
        sqlbuilder.CreateTable("posts").
            IfNotExists().
            Column(sqlbuilder.Column("id", "BIGINT UNSIGNED").NotNull().AutoIncrement()).
            Column(sqlbuilder.Column("title", "VARCHAR(255)").NotNull()).
            Column(sqlbuilder.Column("created_at", "TIMESTAMP").NotNull().DefaultRaw("CURRENT_TIMESTAMP")).
            PrimaryKey("id").
            Engine("InnoDB").
            Collate("utf8mb4_unicode_ci"),
    )
    return err
}
```

See `examples/with_builder/main.go` and `examples/with_connect_and_builder/main.go`.

## Generator

```go
err := migrations.GenerateMigration(migrations.GenerateOptions{
    Dir:         "migrations",
    PackageName: "migrations",
    Name:        "create_users_table",
})
```

Generates `migrations/20260608120000_create_users_table.go`.

To regenerate the migration list file:

```go
migrations.GenerateMigrationList("migrations", "migrations", []string{
    "CreateUsersTable20260608120000",
})
```

## CLI

`cmd/migrations/main.go` is a **skeleton** — it ships with an empty `migrationList`. It is intended as a starting point that you copy into your own application and wire up with your concrete migration list:

```go
// In your copy of cmd/migrations/main.go:
var migrationList []migrations.Migration = yourmigrations.List
```

```bash
export MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/db?parseTime=true'

migrations init                    # create schema_migrations table
migrations create create_users     # generate a new migration file in $MIGRATIONS_DIR
migrations up                      # apply all pending migrations
migrations up --steps=1            # apply exactly 1 migration
migrations down --steps=1          # roll back 1 migration
migrations status                  # print migration status
migrations validate                # validate checksums
```

For a fully wired example (generator + list + CLI) see `examples/generated_migrations/`.

## schema_migrations table

```sql
CREATE TABLE IF NOT EXISTS `schema_migrations` (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

On `Down`, the row is **deleted** to keep the table in sync with the actual schema state. Audit logging should be added at the application level if required.

## Locking

```sql
SELECT GET_LOCK('migrations_lock', 30)
SELECT RELEASE_LOCK('migrations_lock')
```

**MySQL `GET_LOCK` is connection-scoped**: the lock belongs to the connection that called it and is automatically released when that connection closes or is killed. The runner obtains a *dedicated* `*sql.Conn` for the lock and keeps it alive until `RELEASE_LOCK` is called at the end of `Up`/`Down`. This means the connection pool must allow at least `MaxOpenConns >= 2`.

Release is always called via `defer`. A release error is logged but does not mask the primary error. Lock can be disabled: `WithLock(false)`.

## Transactions

| Mode                      | Behaviour                                                                           |
|---------------------------|-------------------------------------------------------------------------------------|
| `TransactionPerMigration` | Each `TxMigration` runs in its own `BEGIN`/`COMMIT` (default)                       |
| `TransactionAll`          | All pending migrations share one `BEGIN`/`COMMIT`. Every migration must implement `TxMigration`; a plain `Migration` causes an error. |
| `TransactionNone`         | No `BEGIN`/`COMMIT` is created by the runner. `Migration.Up/Down` is called directly on `*sql.DB`. |

> **Warning — MySQL DDL cannot be rolled back.** `CREATE TABLE`, `ALTER TABLE`, `DROP TABLE`, and `TRUNCATE` cause an implicit commit in MySQL. They execute immediately and **cannot be undone by a transaction rollback**, including `TransactionAll`. `TransactionAll` is safe only for pure-DML migrations. For DDL migrations, use `TransactionPerMigration` (default) or `TransactionNone`.

## Checksum

There are two checksum strategies:

**Default checksum** (migrations that do NOT implement `ChecksumMigration`):
`sha256(version + "\x00" + name)`. This detects renames and version changes but **does not detect SQL body edits**.

**Generated / SQL-body checksum** (migrations created by the generator):
The generator produces `Checksum()` that computes `sha256(upSQL + "\x00" + downSQL)` from package-level SQL constants. Editing the SQL constant changes the checksum, which triggers `ErrChecksumMismatch` before the modified migration can be silently re-applied.

```go
// Generated migration (simplified):
const createUsersTableUpSQL = `CREATE TABLE users ...`
const createUsersTableDownSQL = `DROP TABLE IF EXISTS users`

func (m CreateUsersTable20260608120000) Checksum() string {
    h := sha256.New()
    h.Write([]byte(createUsersTableUpSQL))
    h.Write([]byte{0})
    h.Write([]byte(createUsersTableDownSQL))
    return fmt.Sprintf("%x", h.Sum(nil))
}
```

For hand-written migrations, implement `ChecksumMigration` to opt in to SQL-body detection:

```go
func (m MyMigration) Checksum() string {
    return "sha256:<hash-of-sql-body>"
}
```

If `Checksum()` returns an empty string, the default `sha256(version+name)` is used as fallback.

Use `WithAllowDirty(true)` to skip checksum validation (not recommended in production).

## Dirty state

If a migration's `Up` or `Down` returns an error, the runner records a **dirty row** (`success=0`) in `schema_migrations`. Subsequent `Up` / `Down` / `Validate` calls return `ErrDirtyState` until the dirty record is resolved.

Recovery options:
- Fix the root cause and retry with `WithAllowDirty(true)` — on success the dirty row is cleared automatically via `INSERT ... ON DUPLICATE KEY UPDATE`.
- Manually delete the dirty row from `schema_migrations` if the migration was partially applied and you want to start over.

## Rollback

`Down` runs the migration's `Down` / `DownTx` and then deletes the record from `schema_migrations`. If `Down` fails, the row is marked dirty (`success=0`, `direction='down'`). Recovery requires manual intervention.

## Errors

```go
var (
    ErrMigrationNotFound       = errors.New("migration not found")
    ErrMigrationAlreadyApplied = errors.New("migration already applied")
    ErrMigrationNotApplied     = errors.New("migration not applied")
    ErrChecksumMismatch        = errors.New("migration checksum mismatch")
    ErrLockNotAcquired         = errors.New("migration lock not acquired")
    ErrDirtyState              = errors.New("migration dirty state")       // blocked by a failed migration
    ErrInvalidIdentifier       = errors.New("invalid identifier")
    ErrDuplicateVersion        = errors.New("duplicate migration version")
    ErrUnsupportedDialect      = errors.New("unsupported dialect")         // WithDialect: only MySQL accepted
    ErrInvalidMigration        = errors.New("invalid migration")           // WithMigrations: nil/empty version/name
)
```

All errors are wrapped with `%w`, so `errors.Is` works at any depth.

## Logger

```go
type Logger interface {
    Printf(format string, args ...any)
}
```

`*log.Logger` satisfies this interface. Default is a noop logger.

## Production recommendations

See `docs/production.md`.
