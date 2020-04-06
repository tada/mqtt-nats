package mqtt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type Reader struct {
	io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r}
}

func (r *Reader) ReadByte() (byte, error) {
	b := []byte{0}
	n, err := r.Read(b)
	if n == 1 {
		return b[0], nil
	}
	return 0, err
}

// ReadVarInt returns the next variable size unsigned integer from the input stream.
// An io.ErrUnexpectedEOF is returned if an io.EOF is encountered before the integer could be fully read.
// A "malformed compressed int" error is returned if the value is larger than 0x0FFFFFFF.
func (r *Reader) ReadVarInt() (int, error) {
	m := 1
	v := 0
	for {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return 0, err
		}
		v += int(b&0x7f) * m
		if (b & 0x80) == 0 {
			return v, nil
		}
		m *= 0x80
		if m > 0x200000 {
			return 0, errors.New("malformed compressed int")
		}
	}
}

func (r *Reader) ReadUint16() (uint16, error) {
	var v uint16
	bs, err := r.ReadExact(2)
	if err == nil {
		v = binary.BigEndian.Uint16(bs)
	}
	return v, err
}

// ReadString will reads a big endian uint16 from the stream that denotes the number of bytes
// that will follow. It then reads those bytes and returns them as a UTF8 encoded string.
func (r *Reader) ReadString() (string, error) {
	var s string
	bs, err := r.ReadBytes()
	if err == nil {
		s = string(bs)
	}
	return s, err
}

// ReadBytes will reads a big endian uint16 from the stream that denotes the number of bytes
// that will follow. It then reads those bytes and returns them.
func (r *Reader) ReadBytes() ([]byte, error) {
	var bs []byte
	l, err := r.ReadUint16()
	if l > 0 && err == nil {
		bs, err = r.ReadExact(int(l))
	}
	return bs, err
}

func (r *Reader) ReadExact(n int) ([]byte, error) {
	bs := make([]byte, n)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		bs = nil
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}
	return bs, err
}

func (r *Reader) Len() int {
	if br, ok := r.Reader.(*bytes.Reader); ok {
		return br.Len()
	}

	// Reader was not set up to read remaining length
	panic(fmt.Errorf("unsupported operation on %T: Len", r.Reader))
}

func (r *Reader) ReadRemainingBytes() ([]byte, error) {
	return r.ReadExact(r.Len())
}

func (r *Reader) ReadPacket(pkLen int) (*Reader, error) {
	// Do a bulk read and switch to read from that bulk
	var rdr *Reader
	pk, err := r.ReadExact(pkLen)
	if err == nil {
		rdr = &Reader{bytes.NewReader(pk)}
	}
	return rdr, err
}
