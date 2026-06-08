package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// Migrator applies and rolls back database migrations.
type Migrator struct {
	db    *sql.DB
	opts  options
	store Store
	lock  Lock
}

// NewMigrator creates a new Migrator backed by the given *sql.DB.
// Migrations are sorted by Version and checked for duplicates before returning.
func NewMigrator(db *sql.DB, optFuncs ...Option) (*Migrator, error) {
	if db == nil {
		return nil, fmt.Errorf("migrations: db must not be nil")
	}

	o := defaultOptions()
	for _, fn := range optFuncs {
		if err := fn(&o); err != nil {
			return nil, fmt.Errorf("migrations: option: %w", err)
		}
	}

	sort.Slice(o.migrations, func(i, j int) bool {
		return o.migrations[i].Version() < o.migrations[j].Version()
	})

	if err := checkDuplicateVersions(o.migrations); err != nil {
		return nil, err
	}

	store := newMySQLStore(db, o.tableName)

	var lock Lock
	if o.lockEnabled {
		lock = newMySQLLock(db, o.lockName, o.lockTimeout)
	} else {
		lock = noopLock{}
	}

	return &Migrator{db: db, opts: o, store: store, lock: lock}, nil
}

func checkDuplicateVersions(ms []Migration) error {
	seen := make(map[string]struct{}, len(ms))
	for _, m := range ms {
		v := m.Version()
		if _, ok := seen[v]; ok {
			return fmt.Errorf("version %q: %w", v, ErrDuplicateVersion)
		}
		seen[v] = struct{}{}
	}
	return nil
}

// Init creates the migration history table if AutoCreateTable is enabled.
func (m *Migrator) Init(ctx context.Context) error {
	if !m.opts.autoCreateTable {
		return nil
	}
	return m.store.CreateTable(ctx)
}

// Up applies all pending migrations.
func (m *Migrator) Up(ctx context.Context) error {
	return m.UpSteps(ctx, 0)
}

// UpSteps applies up to steps pending migrations.
// If steps <= 0 all pending migrations are applied.
func (m *Migrator) UpSteps(ctx context.Context, steps int) error {
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.releaseLock(ctx)

	pending, err := m.pendingMigrations(ctx)
	if err != nil {
		return err
	}

	if steps > 0 && len(pending) > steps {
		pending = pending[:steps]
	}

	return m.applyUp(ctx, pending)
}

// Down rolls back the most recently applied migration.
func (m *Migrator) Down(ctx context.Context) error {
	return m.DownSteps(ctx, 1)
}

// DownSteps rolls back the last steps applied migrations.
// If steps <= 0 it defaults to 1.
func (m *Migrator) DownSteps(ctx context.Context, steps int) error {
	if steps <= 0 {
		steps = 1
	}

	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.releaseLock(ctx)

	applied, err := m.store.Applied(ctx)
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		return nil
	}
	if steps > len(applied) {
		steps = len(applied)
	}

	// Applied is sorted ASC; roll back from the tail.
	toRollback := applied[len(applied)-steps:]
	for i := len(toRollback) - 1; i >= 0; i-- {
		mig, err := m.findMigration(toRollback[i].Version)
		if err != nil {
			return err
		}
		if err := m.runOneDown(ctx, mig); err != nil {
			return err
		}
	}
	return nil
}

// Status returns the current status of every registered migration.
func (m *Migrator) Status(ctx context.Context) ([]StatusItem, error) {
	applied, err := m.store.Applied(ctx)
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[string]AppliedMigration, len(applied))
	for _, a := range applied {
		appliedMap[a.Version] = a
	}

	items := make([]StatusItem, 0, len(m.opts.migrations))
	for _, mig := range m.opts.migrations {
		item := StatusItem{
			Version:  mig.Version(),
			Name:     mig.Name(),
			Checksum: Checksum(mig),
		}
		if a, ok := appliedMap[mig.Version()]; ok {
			item.Applied = true
			item.AppliedAt = a.AppliedAt
			item.ExecutionTime = a.ExecutionTime
		}
		items = append(items, item)
	}
	return items, nil
}

// Pending returns all migrations that have not been applied yet.
func (m *Migrator) Pending(ctx context.Context) ([]Migration, error) {
	return m.pendingMigrations(ctx)
}

// Applied returns all migrations recorded in the history store.
func (m *Migrator) Applied(ctx context.Context) ([]AppliedMigration, error) {
	return m.store.Applied(ctx)
}

// Validate checks that every applied migration still matches its recorded checksum.
func (m *Migrator) Validate(ctx context.Context) error {
	applied, err := m.store.Applied(ctx)
	if err != nil {
		return err
	}

	known := make(map[string]Migration, len(m.opts.migrations))
	for _, mig := range m.opts.migrations {
		known[mig.Version()] = mig
	}

	for _, a := range applied {
		mig, ok := known[a.Version]
		if !ok {
			return fmt.Errorf("version %s: %w", a.Version, ErrMigrationNotFound)
		}
		if Checksum(mig) != a.Checksum {
			return fmt.Errorf("version %s: %w", a.Version, ErrChecksumMismatch)
		}
	}
	return nil
}

// ─── internals ───────────────────────────────────────────────────────────────

func (m *Migrator) releaseLock(ctx context.Context) {
	if err := m.lock.Release(ctx); err != nil {
		m.opts.logger.Printf("migrations: release lock: %v", err)
	}
}

func (m *Migrator) pendingMigrations(ctx context.Context) ([]Migration, error) {
	applied, err := m.store.Applied(ctx)
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[string]AppliedMigration, len(applied))
	for _, a := range applied {
		appliedMap[a.Version] = a
	}

	if m.opts.checksumValidation && !m.opts.allowDirty {
		for _, a := range applied {
			for _, mig := range m.opts.migrations {
				if mig.Version() == a.Version && Checksum(mig) != a.Checksum {
					return nil, fmt.Errorf("version %s: %w", a.Version, ErrChecksumMismatch)
				}
			}
		}
	}

	var pending []Migration
	for _, mig := range m.opts.migrations {
		if _, ok := appliedMap[mig.Version()]; !ok {
			pending = append(pending, mig)
		}
	}
	return pending, nil
}

func (m *Migrator) findMigration(version string) (Migration, error) {
	for _, mig := range m.opts.migrations {
		if mig.Version() == version {
			return mig, nil
		}
	}
	return nil, fmt.Errorf("version %s: %w", version, ErrMigrationNotFound)
}

func (m *Migrator) applyUp(ctx context.Context, pending []Migration) error {
	if m.opts.transactionMode == TransactionAll {
		return m.applyUpInTx(ctx, pending)
	}
	for _, mig := range pending {
		if err := m.runOneUp(ctx, mig); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) runOneUp(ctx context.Context, mig Migration) error {
	m.opts.logger.Printf("migrations: applying %s %s", mig.Version(), mig.Name())
	start := time.Now()

	var runErr error
	if txm, ok := mig.(TxMigration); ok {
		tx, err := m.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("up %s: begin tx: %w", mig.Version(), err)
		}
		if runErr = txm.UpTx(ctx, tx); runErr != nil {
			_ = tx.Rollback()
		} else if runErr = tx.Commit(); runErr != nil {
			runErr = fmt.Errorf("commit: %w", runErr)
		}
	} else {
		runErr = mig.Up(ctx, m.db)
	}

	elapsed := time.Since(start)

	if runErr != nil {
		m.opts.logger.Printf("migrations: failed %s: %v (%s)", mig.Version(), runErr, elapsed)
		return fmt.Errorf("up %s: %w", mig.Version(), runErr)
	}

	if err := m.store.Insert(ctx, AppliedMigration{
		Version:       mig.Version(),
		Name:          mig.Name(),
		Checksum:      Checksum(mig),
		ExecutionTime: elapsed,
	}); err != nil {
		return err
	}

	m.opts.logger.Printf("migrations: applied %s %s (%s)", mig.Version(), mig.Name(), elapsed)
	return nil
}

func (m *Migrator) applyUpInTx(ctx context.Context, pending []Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrations: begin tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	starts := make([]time.Time, len(pending))
	for i, mig := range pending {
		starts[i] = time.Now()
		m.opts.logger.Printf("migrations: applying %s %s", mig.Version(), mig.Name())

		var runErr error
		if txm, ok := mig.(TxMigration); ok {
			runErr = txm.UpTx(ctx, tx)
		} else {
			runErr = mig.Up(ctx, m.db)
		}

		if runErr != nil {
			m.opts.logger.Printf("migrations: failed %s: %v", mig.Version(), runErr)
			return fmt.Errorf("up %s: %w", mig.Version(), runErr)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrations: commit: %w", err)
	}
	committed = true

	for i, mig := range pending {
		if err := m.store.Insert(ctx, AppliedMigration{
			Version:       mig.Version(),
			Name:          mig.Name(),
			Checksum:      Checksum(mig),
			ExecutionTime: time.Since(starts[i]),
		}); err != nil {
			return err
		}
		m.opts.logger.Printf("migrations: applied %s %s", mig.Version(), mig.Name())
	}
	return nil
}

func (m *Migrator) runOneDown(ctx context.Context, mig Migration) error {
	m.opts.logger.Printf("migrations: rolling back %s %s", mig.Version(), mig.Name())
	start := time.Now()

	var runErr error
	if txm, ok := mig.(TxMigration); ok {
		tx, err := m.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("down %s: begin tx: %w", mig.Version(), err)
		}
		if runErr = txm.DownTx(ctx, tx); runErr != nil {
			_ = tx.Rollback()
		} else if runErr = tx.Commit(); runErr != nil {
			runErr = fmt.Errorf("commit: %w", runErr)
		}
	} else {
		runErr = mig.Down(ctx, m.db)
	}

	elapsed := time.Since(start)

	if runErr != nil {
		m.opts.logger.Printf("migrations: rollback failed %s: %v (%s)", mig.Version(), runErr, elapsed)
		return fmt.Errorf("down %s: %w", mig.Version(), runErr)
	}

	if err := m.store.Delete(ctx, mig.Version()); err != nil {
		return err
	}

	m.opts.logger.Printf("migrations: rolled back %s %s (%s)", mig.Version(), mig.Name(), elapsed)
	return nil
}
