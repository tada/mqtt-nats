// Package logger contains a logger interface and an implementation that is based ont he standard log.Logger
package logger

import (
	"io"
	"log"
)

// A Logger logs information using a log level
type Logger interface {
	// DebugEnabled returns true if debug level logging is enabled
	DebugEnabled() bool

	// Debug logs at debug level. Arguments are handled in the manner of fmt.Println.
	Debug(...interface{})

	// ErrorEnabled returns true if error level logging is enabled
	ErrorEnabled() bool

	// Error logs at error level. Arguments are handled in the manner of fmt.Println.
	Error(...interface{})

	// InfoEnabled returns true if info level logging is enabled
	InfoEnabled() bool

	// Info logs at info level. Arguments are handled in the manner of fmt.Println.
	Info(...interface{})
}

// Level determines at of logging that is enabled
type Level int

const (
	// Silent means that all logging is disabled
	Silent = Level(iota)

	// Error means that only error logging is enabled
	Error

	// Info means that error and info logging is enabled
	Info

	// Debug means that all logging is enabled
	Debug
)

type silent int

func (silent) Debug(...interface{}) {
}
func (silent) DebugEnabled() bool {
	return false
}
func (silent) Error(...interface{}) {
}
func (silent) ErrorEnabled() bool {
	return false
}
func (silent) Info(...interface{}) {
}
func (silent) InfoEnabled() bool {
	return false
}

type writer struct {
	debug *log.Logger
	info  *log.Logger
	err   *log.Logger
}

func (l *writer) Debug(args ...interface{}) {
	if l.debug != nil {
		l.debug.Println(args...)
	}
}

func (l *writer) DebugEnabled() bool {
	return l.debug != nil
}

func (l *writer) Error(args ...interface{}) {
	if l.err != nil {
		l.err.Println(args...)
	}
}

func (l *writer) ErrorEnabled() bool {
	return l.err != nil
}

func (l *writer) Info(args ...interface{}) {
	if l.info != nil {
		l.info.Println(args...)
	}
}

func (l *writer) InfoEnabled() bool {
	return l.info != nil
}

// New returns a logger that is based on the standard log.Logger.
func New(level Level, out, err io.Writer) Logger {
	if level == Silent {
		return silent(0)
	}

	l := &writer{}
	switch level {
	case Debug:
		l.debug = log.New(out, "DEBUG ", log.LstdFlags)
		fallthrough
	case Info:
		l.info = log.New(out, "INFO  ", log.LstdFlags)
		fallthrough
	case Error:
		l.err = log.New(err, "ERROR ", log.LstdFlags)
	}
	return l
}
