package pio

import (
	"bytes"
	"errors"
	"testing"
)

func TestWriteBool(t *testing.T) {
	w := bytes.Buffer{}
	WriteBool(true, &w)
	WriteBool(false, &w)
	a := w.String()
	if a != "truefalse" {
		t.Error("expected 'truefalse', got", a)
	}
}

func TestWriteFloat(t *testing.T) {
	w := bytes.Buffer{}
	WriteFloat(3.14159, &w)
	a := w.String()
	if a != "3.14159" {
		t.Error("expected '3.14159', got", a)
	}
}

func TestWriteRune(t *testing.T) {
	w := bytes.Buffer{}
	WriteRune('⌘', &w)
	a := w.String()
	if a != "⌘" {
		t.Error("expected '⌘', got", a)
	}
}

type badWriter int

func (b badWriter) Write(p []byte) (n int, err error) {
	return 0, &Error{Cause: errors.New("bad write")}
}

func TestWriteError(t *testing.T) {
	err := Catch(func() error {
		WriteRune('⌘', badWriter(0))
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
