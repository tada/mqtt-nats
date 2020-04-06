package testutils

import (
	"testing"

	"github.com/tada/mqtt-nats/logger"
)

type mockT struct {
	logEntries [][]interface{}
}

func (m *mockT) Helper() {
}

func (m *mockT) Log(args ...interface{}) {
	m.logEntries = append(m.logEntries, args)
}

func Test_testLogger_Debug(t *testing.T) {
	tf := mockT{}
	lg := NewLogger(logger.Debug, &tf)
	CheckTrue(lg.DebugEnabled(), t)
	CheckTrue(lg.InfoEnabled(), t)
	CheckTrue(lg.ErrorEnabled(), t)
	lg.Debug("some stuff")
	CheckEqual(1, len(tf.logEntries), t)
	le := tf.logEntries[0]
	CheckEqual(2, len(le), t)
	CheckEqual("DEBUG", le[0], t)
	CheckEqual("some stuff", le[1], t)
}

func Test_testLogger_Info(t *testing.T) {
	tf := mockT{}
	lg := NewLogger(logger.Info, &tf)
	CheckFalse(lg.DebugEnabled(), t)
	CheckTrue(lg.InfoEnabled(), t)
	CheckTrue(lg.ErrorEnabled(), t)
	lg.Info("some stuff")
	CheckEqual(1, len(tf.logEntries), t)
	le := tf.logEntries[0]
	CheckEqual(2, len(le), t)
	CheckEqual("INFO ", le[0], t)
	CheckEqual("some stuff", le[1], t)
}

func Test_testLogger_Error(t *testing.T) {
	tf := mockT{}
	lg := NewLogger(logger.Error, &tf)
	CheckFalse(lg.DebugEnabled(), t)
	CheckFalse(lg.InfoEnabled(), t)
	CheckTrue(lg.ErrorEnabled(), t)
	lg.Error("some stuff")
	CheckEqual(1, len(tf.logEntries), t)
	le := tf.logEntries[0]
	CheckEqual(2, len(le), t)
	CheckEqual("ERROR", le[0], t)
	CheckEqual("some stuff", le[1], t)
}
