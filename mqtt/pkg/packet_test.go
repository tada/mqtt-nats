package pkg

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tada/mqtt-nats/mqtt"
)

func writeReadAndCompare(t *testing.T, p Packet, ex string) {
	w := &mqtt.Writer{}
	p.Write(w)
	p2 := Parse(t, bytes.NewReader(w.Bytes()))
	if !p.Equals(p2) {
		t.Fatal(p, "!=", p2)
	}
	ac := p.(fmt.Stringer).String()
	if ex != ac {
		t.Errorf("expected '%s' got '%s'", ex, ac)
	}
}
