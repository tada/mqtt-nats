package mqtt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Reader extends the io.Reader with MQTT specific semantics for reading variable length integers,
// two byte unsigned integers, and length prefixed strings and bytes.
type Reader struct {
	io.Reader
}

// NewReader creates a new Reader that reads from the given io.Reader
func NewReader(r io.Reader) *Reader {
	return &Reader{r}
}

// ReadByte reads and returns the next byte from the input or
// any error encountered. If ReadByte returns an error, no input
// byte was consumed, and the returned byte value is undefined.
func (r *Reader) ReadByte() (byte, error) {
	b := []byte{0}
	n, err := r.Read(b)
	if n == 1 {
		return b[0], nil
	}
	return 0, err
}

// ReadVarInt returns the next variable size unsigned integer from the input stream.
// A "malformed compressed int" error is returned if the value is larger than 0x0FFFFFFF.
//
// An io.ErrUnexpectedEOF is returned if EOF is encountered during the read.
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

// ReadUint16 reads the next two bytes from the input stream and returns a big endian unsigned integer
//
// An io.ErrUnexpectedEOF is returned if EOF is encountered during the read.
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
//
// An io.ErrUnexpectedEOF is returned if EOF is encountered during the read.
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
//
// An io.ErrUnexpectedEOF is returned if EOF is encountered during the read.
func (r *Reader) ReadBytes() ([]byte, error) {
	var bs []byte
	l, err := r.ReadUint16()
	if l > 0 && err == nil {
		bs, err = r.ReadExact(int(l))
	}
	return bs, err
}

// ReadExact reads an exact number of bytes into a []byte slice and returns it.
//
// An io.ErrUnexpectedEOF is returned if EOF is encountered during the read.
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

// Len returns the number of bytes of the unread portion of the slice. This method will panic
// unless the underlying reader is a bytes.Reader.
func (r *Reader) Len() int {
	if br, ok := r.Reader.(*bytes.Reader); ok {
		return br.Len()
	}

	// Reader was not set up to read remaining length
	panic(fmt.Errorf("unsupported operation on %T: Len", r.Reader))
}

// ReadRemainingBytes returns the remaining bytes of the underlying reader. This method will panic
// unless the underlying reader is a bytes.Reader.
func (r *Reader) ReadRemainingBytes() ([]byte, error) {
	return r.ReadExact(r.Len())
}

// ReadPacket reads exactly pkLen bytes from the underlying reader into a byte slice and creates a new Reader
// that will read from this slice. The new Reader is returned.
//
// An io.ErrUnexpectedEOF is returned if EOF is encountered during the read.
func (r *Reader) ReadPacket(pkLen int) (*Reader, error) {
	// Do a bulk read and switch to read from that bulk
	var rdr *Reader
	pk, err := r.ReadExact(pkLen)
	if err == nil {
		rdr = &Reader{bytes.NewReader(pk)}
	}
	return rdr, err
}
