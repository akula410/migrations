package migrations

import (
	"fmt"
	"regexp"
	"time"
)

// Dialect identifies the target database engine.
type Dialect string

const (
	MySQL Dialect = "mysql"
)

// TransactionMode controls how the runner wraps migrations in transactions.
type TransactionMode int

const (
	// TransactionPerMigration wraps each TxMigration in its own transaction (default).
	TransactionPerMigration TransactionMode = iota
	// TransactionAll wraps all TxMigrations in a single shared transaction.
	TransactionAll
	// TransactionNone disables all transaction wrapping. The runner calls Migration.Up/Down
	// directly on *sql.DB without creating any *sql.Tx, even for TxMigration implementations.
	TransactionNone
)

var reValidIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func validateIdentifier(name string) error {
	if !reValidIdentifier.MatchString(name) {
		return fmt.Errorf("%q: %w", name, ErrInvalidIdentifier)
	}
	return nil
}

type options struct {
	migrations         []Migration
	tableName          string
	dialect            Dialect
	logger             Logger
	lockEnabled        bool
	lockName           string
	lockTimeout        time.Duration
	autoCreateTable    bool
	transactionMode    TransactionMode
	checksumValidation bool
	allowDirty         bool
}

func defaultOptions() options {
	return options{
		tableName:          "schema_migrations",
		dialect:            MySQL,
		logger:             noopLogger{},
		lockEnabled:        true,
		lockName:           "migrations_lock",
		lockTimeout:        30 * time.Second,
		autoCreateTable:    true,
		transactionMode:    TransactionPerMigration,
		checksumValidation: true,
		allowDirty:         false,
	}
}

// Option is a functional option for NewMigrator.
type Option func(*options) error

// WithMigrations adds migrations to the migrator.
// Returns ErrInvalidMigration if any migration is nil or has an empty Version or Name.
func WithMigrations(ms ...Migration) Option {
	return func(o *options) error {
		for _, m := range ms {
			if m == nil {
				return fmt.Errorf("WithMigrations: nil migration: %w", ErrInvalidMigration)
			}
			if m.Version() == "" {
				return fmt.Errorf("WithMigrations: empty version: %w", ErrInvalidMigration)
			}
			if m.Name() == "" {
				return fmt.Errorf("WithMigrations: empty name: %w", ErrInvalidMigration)
			}
		}
		o.migrations = append(o.migrations, ms...)
		return nil
	}
}

// WithTableName sets the migration history table name.
// Must match ^[a-zA-Z_][a-zA-Z0-9_]*$.
func WithTableName(name string) Option {
	return func(o *options) error {
		if err := validateIdentifier(name); err != nil {
			return fmt.Errorf("WithTableName: %w", err)
		}
		o.tableName = name
		return nil
	}
}

// WithDialect sets the database dialect. Only MySQL is supported; any other value
// returns ErrUnsupportedDialect.
func WithDialect(dialect Dialect) Option {
	return func(o *options) error {
		if dialect != MySQL {
			return fmt.Errorf("WithDialect %q: %w", dialect, ErrUnsupportedDialect)
		}
		o.dialect = dialect
		return nil
	}
}

// WithLogger sets a logger for migration progress output.
func WithLogger(logger Logger) Option {
	return func(o *options) error {
		if logger != nil {
			o.logger = logger
		}
		return nil
	}
}

// WithLock enables or disables distributed locking via MySQL GET_LOCK.
func WithLock(enabled bool) Option {
	return func(o *options) error {
		o.lockEnabled = enabled
		return nil
	}
}

// WithLockName sets the lock name used in GET_LOCK/RELEASE_LOCK.
// Must match ^[a-zA-Z_][a-zA-Z0-9_]*$.
func WithLockName(name string) Option {
	return func(o *options) error {
		if err := validateIdentifier(name); err != nil {
			return fmt.Errorf("WithLockName: %w", err)
		}
		o.lockName = name
		return nil
	}
}

// WithLockTimeout sets the timeout for acquiring the lock.
func WithLockTimeout(timeout time.Duration) Option {
	return func(o *options) error {
		o.lockTimeout = timeout
		return nil
	}
}

// WithAutoCreateTable controls automatic creation of the migrations history table.
func WithAutoCreateTable(enabled bool) Option {
	return func(o *options) error {
		o.autoCreateTable = enabled
		return nil
	}
}

// WithTransactionMode sets the transaction strategy for migration execution.
func WithTransactionMode(mode TransactionMode) Option {
	return func(o *options) error {
		o.transactionMode = mode
		return nil
	}
}

// WithChecksumValidation enables or disables checksum mismatch detection.
func WithChecksumValidation(enabled bool) Option {
	return func(o *options) error {
		o.checksumValidation = enabled
		return nil
	}
}

// WithAllowDirty skips checksum mismatch errors when set to true.
func WithAllowDirty(enabled bool) Option {
	return func(o *options) error {
		o.allowDirty = enabled
		return nil
	}
}
