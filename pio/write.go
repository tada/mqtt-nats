package pio

import (
	"io"
	"strconv"
	"unicode/utf8"
)

// Write writes the bytes b to the Writer, returning its length.
//
// If an error occurs the method panics with a Error with the Cause set to that error
func Write(b []byte, w io.Writer) int {
	n, err := w.Write(b)
	if err != nil {
		panic(&Error{err})
	}
	return n
}

// WriteString writes the bytes of s to the Writer, returning its length.
//
// If an error occurs the method panics with a Error with the Cause set to that error
func WriteString(s string, w io.Writer) int {
	return Write([]byte(s), w)
}

// WriteByte writes the byte r to the Writer.
//
// If an error occurs the method panics with a Error with the Cause set to that error
func WriteByte(b byte, w io.Writer) {
	Write([]byte{b}, w)
}

// WriteRune writes the UTF-8 encoding of Unicode code point r to the Writer, returning its length.
//
// If an error occurs the method panics with a Error with the Cause set to that error
func WriteRune(r rune, w io.Writer) int {
	if r < utf8.RuneSelf {
		WriteByte(byte(r), w)
		return 1
	}
	b := make([]byte, utf8.UTFMax)
	n := utf8.EncodeRune(b, r)
	Write(b[:n], w)
	return n
}

// WriteBool writes the string "true" or "false" onto the stream
//
// If an error occurs the method panics with a Error with the Cause set to that error
func WriteBool(b bool, w io.Writer) {
	s := "false"
	if b {
		s = "true"
	}
	WriteString(s, w)
}

// WriteInt writes decimal string representation of the given integer onto the stream
//
// If an error occurs the method panics with a Error with the Cause set to that error
func WriteInt(i int64, w io.Writer) {
	WriteString(strconv.FormatInt(i, 10), w)
}

// WriteFloat writes the "%g" string representation of the given integer onto the stream
//
// If an error occurs the method panics with a Error with the Cause set to that error
func WriteFloat(f float64, w io.Writer) {
	WriteString(strconv.FormatFloat(f, 'g', -1, 64), w)
}
