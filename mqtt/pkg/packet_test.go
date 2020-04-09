package pkg_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tada/mqtt-nats/test/packet"

	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/mqtt/pkg"
)

func writeReadAndCompare(t *testing.T, p pkg.Packet, ex string) {
	w := &mqtt.Writer{}
	p.Write(w)
	p2 := packet.Parse(t, bytes.NewReader(w.Bytes()))
	if !p.Equals(p2) {
		t.Fatal(p, "!=", p2)
	}
	ac := p.(fmt.Stringer).String()
	if ex != ac {
		t.Errorf("expected '%s' got '%s'", ex, ac)
	}
}
