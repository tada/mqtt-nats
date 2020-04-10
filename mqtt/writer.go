package mqtt

import "bytes"

// Writer extends a bytes.Buffer with MQTT specific semantics for writing variable length integers,
// two byte unsigned integers, and length prefixed strings and bytes.
type Writer struct {
	bytes.Buffer
}

// NewWriter returns a new Writer instance
func NewWriter() *Writer {
	return &Writer{}
}

// WriteU8 writes a byte on the underlying buffer. This is the same as calling WriteByte but there
// is no error return as opposed to WriteByte which returns the error type although it is always nil.
func (w *Writer) WriteU8(i uint8) {
	_ = w.WriteByte(i)
}

// WriteU16 writes the big endian two bytes of the given uint16 on the underlying buffer
func (w *Writer) WriteU16(i uint16) {
	w.WriteU8(byte(i >> 8))
	w.WriteU8(byte(i))
}

// WriteString first writes the length of the string using WriteU16 and then the strings bytes.
func (w *Writer) WriteString(s string) {
	t := len(s)
	w.WriteU16(uint16(t))
	for i := 0; i < t; i++ {
		w.WriteU8(s[i])
	}
}

// WriteBytes first writes the length of the byte slice using WriteU16 and then the bytes.
func (w *Writer) WriteBytes(bs []byte) {
	t := len(bs)
	w.WriteU16(uint16(t))
	_, _ = w.Write(bs)
}

// WriteVarInt writes a variable length integer in accordance with the MQTT 3.1.1 specification on the
// underlying buffer.
func (w *Writer) WriteVarInt(value int) {
	for {
		b := byte(value & 0x7f)
		value >>= 7
		if value > 0 {
			b |= 0x80
		}
		w.WriteU8(b)
		if value == 0 {
			break
		}
	}
}
