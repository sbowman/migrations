package migrations

import (
	"fmt"
	"os"
)

var (
	// Log outputs log messages to any object supporting the Logger
	// interface.  Replace at any time to a logger compatible with your
	// system.
	Log Logger
)

func init() {
	Log = new(DefaultLogger)
}

// Logger is a simple interface for logging in migrations.
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
}

// DefaultLogger implements Logger.  It outputs to stdout for debug messages and
// stderr for error messages.
type DefaultLogger struct{}

// Debugf outputs debug messages to stdout.
func (log *DefaultLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
}

// Infof outputs info or error messages to stderr.
func (log *DefaultLogger) Infof(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
	_, _ = fmt.Fprintln(os.Stderr)
}

// NilLogger hides migration logging output.
type NilLogger struct{}

// Debugf outputs nothing.
func (log *NilLogger) Debugf(string, ...interface{}) {
}

// Infof outputs nothing.
func (log *NilLogger) Infof(string, ...interface{}) {
}
