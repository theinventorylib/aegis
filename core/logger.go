package core

// Logger is the minimal logging interface used by core services for
// non-fatal operational events (cache errors, recoverable store errors,
// etc). It is structurally compatible with config.Logger so that the
// application-level logger can be plumbed in without an import cycle.
type Logger interface {
	Info(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	Debug(msg string, keysAndValues ...any)
}

// noopLogger discards all log messages. It is used as a safe default so
// callers never need to nil-check the logger.
type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}
func (noopLogger) Debug(string, ...any) {}
