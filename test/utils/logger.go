// +build citest

package utils

import (
	"github.com/tada/mqtt-nats/logger"
)

// The T interface is fulfilled by *testing.T but can be implemented by a T mock if needed.
type T interface {
	Helper()
	Log(...interface{})
}

// NewLogger creates a new Logger configured to log at the given level. The logger uses the given
// testing.T's Log function for all output.
func NewLogger(l logger.Level, t T) logger.Logger {
	return &testLogger{l: l, t: t}
}

type testLogger struct {
	l logger.Level
	t T
}

func (t *testLogger) DebugEnabled() bool {
	return t.l >= logger.Debug
}

func (t *testLogger) Debug(args ...interface{}) {
	if t.DebugEnabled() {
		t.t.Helper()
		t.t.Log(addFirst("DEBUG", args)...)
	}
}

func (t *testLogger) ErrorEnabled() bool {
	return t.l >= logger.Error
}

func (t *testLogger) Error(args ...interface{}) {
	if t.ErrorEnabled() {
		t.t.Helper()
		t.t.Log(addFirst("ERROR", args)...)
	}
}

func (t *testLogger) InfoEnabled() bool {
	return t.l >= logger.Info
}

func (t *testLogger) Info(args ...interface{}) {
	if t.InfoEnabled() {
		t.t.Helper()
		t.t.Log(addFirst("INFO ", args)...)
	}
}

// addFirst prepends the client to the args slice and returns the new slice
func addFirst(first string, args []interface{}) []interface{} {
	na := make([]interface{}, len(args)+1)
	na[0] = first
	copy(na[1:], args)
	return na
}
