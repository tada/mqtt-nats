package logger

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"
)

func TestLogger(t *testing.T) {
	l := New(Silent, nil, nil)
	if _, ok := l.(silent); !ok {
		t.Error("silent leven did not result in silent logger")
	}
}

var msgEx = regexp.MustCompile(`^(A-Z)\s+^[\s]+\s^[\s+]\s(.*)$`)

func assertEqual(t *testing.T, e, a string) {
	t.Helper()
	if e != a {
		t.Fatalf("expected '%s', got '%s'", e, a)
	}
}

func checkLogOutput(t *testing.T, ls, ms string, bf fmt.Stringer) {
	t.Helper()
	if ps := msgEx.FindStringSubmatch(bf.String()); len(ps) == 2 {
		assertEqual(t, ls, ps[0])
		assertEqual(t, ms, ps[1])
	}
}

func TestLogger_Debug(t *testing.T) {
	o := bytes.Buffer{}
	e := bytes.Buffer{}
	l := New(Debug, &o, &e)
	m := "some message"
	if !(l.ErrorEnabled() && l.InfoEnabled() && l.DebugEnabled()) {
		t.Fatal("wrong levels enabled")
	}
	l.Debug(m)
	if e.Len() > 0 {
		t.Error("debug log produced output on stderr")
	}
	l.Error(m)
	checkLogOutput(t, "DEBUG", m, &o)
	checkLogOutput(t, "ERROR", m, &e)
}

func TestLogger_Info(t *testing.T) {
	o := bytes.Buffer{}
	e := bytes.Buffer{}
	l := New(Info, &o, &e)
	m := "some message"
	if !(l.ErrorEnabled() && l.InfoEnabled() && !l.DebugEnabled()) {
		t.Fatal("wrong levels enabled")
	}
	l.Info(m)
	if e.Len() > 0 {
		t.Error("info log produced output on stderr")
	}
	l.Error(m)
	checkLogOutput(t, "INFO", m, &o)
	checkLogOutput(t, "ERROR", m, &e)
}

func TestLogger_Error(t *testing.T) {
	o := bytes.Buffer{}
	e := bytes.Buffer{}
	l := New(Error, &o, &e)
	m := "some message"
	if !(l.ErrorEnabled() && !l.InfoEnabled() && !l.DebugEnabled()) {
		t.Fatal("wrong levels enabled")
	}
	l.Error(m)
	l.Info(m)
	if o.Len() > 0 {
		t.Error("error log produced output on stdout")
	}
	checkLogOutput(t, "ERROR", m, &e)
}

func TestLogger_Silent(t *testing.T) {
	o := bytes.Buffer{}
	e := bytes.Buffer{}
	l := New(Silent, &o, &e)
	m := "some message"
	if l.ErrorEnabled() || l.InfoEnabled() || l.DebugEnabled() {
		t.Fatal("wrong levels enabled")
	}
	l.Debug(m)
	l.Info(m)
	l.Error(m)
	if o.Len() > 0 {
		t.Error("silent log produced output on stdout")
	}
	if e.Len() > 0 {
		t.Error("silent log produced output on stderr")
	}
}
