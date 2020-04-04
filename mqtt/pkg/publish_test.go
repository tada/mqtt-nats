package pkg

import (
	"testing"

	"github.com/tada/mqtt-nats/jsonstream"
)

func TestParsePublish(t *testing.T) {
	writeReadAndCompare(t, NewPublish(23, "some/topic", 2, []byte(`the "message"`), false, ""), ParsePublish, "PUBLISH (d0, q1, r0, m23, 'some/topic', ... (13 bytes))")
}

func TestParsePubAck(t *testing.T) {
	writeReadAndCompare(t, PubAck(23), ParsePubAck, "PUBACK (m23)")
}

func TestParsePubRec(t *testing.T) {
	writeReadAndCompare(t, PubRec(23), ParsePubRec, "PUBREC (m23)")
}

func TestParsePubRel(t *testing.T) {
	writeReadAndCompare(t, PubRel(23), ParsePubRel, "PUBREL (m23)")
}

func TestParsePubComp(t *testing.T) {
	writeReadAndCompare(t, PubComp(23), ParsePubComp, "PUBCOMP (m23)")
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
