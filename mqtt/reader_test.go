package mqtt

import (
	"bytes"
	"io"
	"testing"
)

type noLen int

func (noLen) Read([]byte) (int, error) {
	return 0, io.EOF
}

func TestReader_Len(t *testing.T) {
	l := NewReader(bytes.NewReader([]byte{1, 2, 3})).Len()
	if l != 3 {
		t.Fatalf("expected length 3 got %d", l)
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
	}()
	NewReader(noLen(0)).Len()
}

func TestReader_ReadBytes(t *testing.T) {
	r := NewReader(bytes.NewReader([]byte{0, 2, 'a', 'b'}))
	bs, err := r.ReadBytes()
	if err != nil {
		t.Fatal(err)
	}
	sbs := string(bs)
	if sbs != "ab" {
		t.Fatalf(`expected "ab", got %q`, sbs)
	}

	// test premature EOF
	r = NewReader(bytes.NewReader([]byte{0, 3, 'a', 'b'}))
	_, err = r.ReadBytes()
	if err != io.ErrUnexpectedEOF {
		t.Fatal("expected error")
	}

	// test premature EOF nothing after length
	r = NewReader(bytes.NewReader([]byte{0, 3}))
	_, err = r.ReadBytes()
	if err != io.ErrUnexpectedEOF {
		t.Fatal("expected error")
	}
}

func TestReader_ReadVarInt(t *testing.T) {
	r := NewReader(bytes.NewReader([]byte{0x82, 0xff, 0x3}))
	l, err := r.ReadVarInt()
	if err != nil {
		t.Fatal(err)
	}
	if l != 0xff82 {
		t.Fatalf("expected length 0xff82 got 0x%x", l)
	}
	_, err = NewReader(bytes.NewReader([]byte{0xff, 0xff, 0xff, 0xff, 0xff})).ReadVarInt()
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = NewReader(bytes.NewReader([]byte{0x80})).ReadVarInt()
	if err != io.ErrUnexpectedEOF {
		t.Fatal("expected error")
	}
}
