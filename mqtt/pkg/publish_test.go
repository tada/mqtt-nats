package pkg

import (
	"bytes"
	"testing"

	"github.com/tada/mqtt-nats/jsonstream"

	"github.com/tada/mqtt-nats/mqtt"
)

func TestParsePublish(t *testing.T) {
	p1 := NewPublish(23, "some/topic", 2, []byte(`the "message"`), false, "")
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
	p1 := NewPublish(23, "some/topic", 2, []byte(`the "message"`), false, "")
	bs, err := jsonstream.Marshal(p1)
	if err != nil {
		t.Fatal(err)
	}

	p2 := &Publish{}
	err = jsonstream.Unmarshal(p2, bs)
	if err != nil {
		t.Fatal(err)
	}
	if !p1.Equals(p2) {
		t.Fatal(p1, "!=", p2)
	}
}

func TestPublish_MarshalToJSON_nonUTF(t *testing.T) {
	p1 := NewPublish(23, "some/topic", 2, []byte{0, 1, 2, 3, 5}, false, "")
	bs, err := jsonstream.Marshal(p1)
	if err != nil {
		t.Fatal(err)
	}

	p2 := &Publish{}
	err = jsonstream.Unmarshal(p2, bs)
	if err != nil {
		t.Fatal(err)
	}
	if !p1.Equals(p2) {
		t.Fatal(p1, "!=", p2)
	}
}
