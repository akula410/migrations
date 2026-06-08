package migrations

// Logger is the minimal interface used by the migrator for progress output.
type Logger interface {
	Printf(format string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Printf(_ string, _ ...any) {}
