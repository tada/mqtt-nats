package pkg

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/tada/mqtt-nats/jsonstream"

	"github.com/tada/mqtt-nats/mqtt"
)

func TestParsePublish(t *testing.T) {
	p1 := NewPublish(23, "some/topic", 2, []byte("the message"), false, "")
	w := &mqtt.Writer{}
	p1.Write(w)

	r := mqtt.NewReader(bytes.NewReader(w.Bytes()))
	b, _ := r.ReadByte()
	pl, _ := r.ReadVarInt()
	p2, err := ParsePublish(r, b, pl)
	if err != nil {
		t.Fatal(err)
	}
	if !p1.Equals(p2) {
		t.Fatal(p1, "!=", p2)
	}
}

func TestPublish_MarshalToJSON(t *testing.T) {
	p1 := NewPublish(23, "some/topic", 2, []byte("the message"), false, "")
	w := &bytes.Buffer{}
	p1.MarshalToJSON(w)

	js := json.NewDecoder(bytes.NewReader(w.Bytes()))
	js.UseNumber()
	p2 := &Publish{}
	jsonstream.AssertConsumer(js, p2)
	if !p1.Equals(p2) {
		t.Fatal(p1, "!=", p2)
	}
}
