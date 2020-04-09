package pkg_test

import (
	"testing"

	"github.com/tada/mqtt-nats/jsonstream"
	"github.com/tada/mqtt-nats/mqtt/pkg"
)

func TestParsePublish(t *testing.T) {
	writeReadAndCompare(t, pkg.NewPublish(23, "some/topic", 2, []byte(`the "message"`), false, ""),
		"PUBLISH (d0, q1, r0, m23, 'some/topic', ... (13 bytes))")
}

func TestParsePubAck(t *testing.T) {
	writeReadAndCompare(t, pkg.PubAck(23), "PUBACK (m23)")
}

func TestParsePubRec(t *testing.T) {
	writeReadAndCompare(t, pkg.PubRec(23), "PUBREC (m23)")
}

func TestParsePubRel(t *testing.T) {
	writeReadAndCompare(t, pkg.PubRel(23), "PUBREL (m23)")
}

func TestParsePubComp(t *testing.T) {
	writeReadAndCompare(t, pkg.PubComp(23), "PUBCOMP (m23)")
}

func TestPublish_MarshalToJSON(t *testing.T) {
	p1 := pkg.NewPublish(23, "some/topic", 2, []byte(`the "message"`), false, "")
	bs, err := jsonstream.Marshal(p1)
	if err != nil {
		t.Fatal(err)
	}

	p2 := &pkg.Publish{}
	err = jsonstream.Unmarshal(p2, bs)
	if err != nil {
		t.Fatal(err)
	}
	if !p1.Equals(p2) {
		t.Fatal(p1, "!=", p2)
	}
}

func TestPublish_MarshalToJSON_nonUTF(t *testing.T) {
	p1 := pkg.NewPublish(23, "some/topic", 2, []byte{0, 1, 2, 3, 5}, false, "")
	bs, err := jsonstream.Marshal(p1)
	if err != nil {
		t.Fatal(err)
	}

	p2 := &pkg.Publish{}
	err = jsonstream.Unmarshal(p2, bs)
	if err != nil {
		t.Fatal(err)
	}
	if !p1.Equals(p2) {
		t.Fatal(p1, "!=", p2)
	}
}
