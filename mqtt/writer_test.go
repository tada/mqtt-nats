package mqtt

import (
	"bytes"
	"testing"
)

func TestWriteVarInt(t *testing.T) {
	ints := []int{
		0, 1, 127, 128, 16383, 16384, 2097151, 2097152, 268435455,
	}
	lens := []int{
		1, 1, 1, 2, 2, 3, 3, 4, 4,
	}
	w := NewWriter()
	tl := 0
	for i, v := range ints {
		w.WriteVarInt(v)
		tl += lens[i]
		if tl != w.Len() {
			t.Fatalf("expected len %d, got %d", tl, w.Len())
		}
	}

	r := Reader{bytes.NewReader(w.Bytes())}
	for _, v := range ints {
		x, _ := r.ReadVarInt()
		if v != x {
			t.Fatalf("expected %d, got %d", v, x)
		}
	}
}
