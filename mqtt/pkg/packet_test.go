package pkg

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tada/mqtt-nats/mqtt"
)

func writeReadAndCompare(t *testing.T, p Packet, parser func(*mqtt.Reader, byte, int) (Packet, error), ex string) {
	w := &mqtt.Writer{}
	p.Write(w)

	r := mqtt.NewReader(bytes.NewReader(w.Bytes()))
	b, err := r.ReadByte()
	if err != nil {
		t.Fatal(err)
	}
	pl, err := r.ReadVarInt()
	if err != nil {
		t.Fatal(err)
	}
	p2, err := parser(r, b, pl)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Equals(p2) {
		t.Fatal(p, "!=", p2)
	}
	ac := p.(fmt.Stringer).String()
	if ex != ac {
		t.Errorf("expected '%s' got '%s'", ex, ac)
	}
}
