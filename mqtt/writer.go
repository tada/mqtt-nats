package mqtt

import "bytes"

type Writer struct {
	bytes.Buffer
}

func NewWriter() *Writer {
	return &Writer{}
}

func (w *Writer) WriteU8(i uint8) {
	_ = w.WriteByte(i)
}

func (w *Writer) WriteU16(i uint16) {
	w.WriteU8(byte(i >> 8))
	w.WriteU8(byte(i))
}

func (w *Writer) WriteString(s string) {
	t := len(s)
	w.WriteU16(uint16(t))
	for i := 0; i < t; i++ {
		w.WriteU8(s[i])
	}
}

func (w *Writer) WriteBytes(bs []byte) {
	t := len(bs)
	w.WriteU16(uint16(t))
	_, _ = w.Write(bs)
}

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
