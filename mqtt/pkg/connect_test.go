package pkg

import (
	"bytes"
	"testing"

	"github.com/tada/mqtt-nats/mqtt"
)

func TestParseConnect(t *testing.T) {
	c1 := NewConnect(`cid`, true, 5, &Will{
		Topic:   "my/will",
		Message: []byte("the will"),
		QoS:     1,
		Retain:  false,
	}, "bob", []byte("password"))
	w := &mqtt.Writer{}
	c1.Write(w)

	r := mqtt.NewReader(bytes.NewReader(w.Bytes()))
	b, _ := r.ReadByte()
	pl, _ := r.ReadVarInt()
	c2, err := ParseConnect(r, b, pl)
	if err != nil {
		t.Fatal(err)
	}
	if !c1.Equals(c2) {
		t.Fatal(c1, "!=", c2)
	}
}
