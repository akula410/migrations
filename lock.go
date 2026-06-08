package migrations

import "context"

// Lock controls distributed locking around migration execution.
type Lock interface {
	Acquire(ctx context.Context) error
	Release(ctx context.Context) error
}

type noopLock struct{}

func (noopLock) Acquire(_ context.Context) error { return nil }
func (noopLock) Release(_ context.Context) error { return nil }
