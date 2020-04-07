package pkg

import (
	"bytes"
	"testing"

	"github.com/tada/mqtt-nats/testutils"

	"github.com/tada/mqtt-nats/mqtt"
)

func TestParseSubscribe(t *testing.T) {
	writeReadAndCompare(t, NewSubscribe(23, Topic{Name: "some/topic", QoS: 1}),
		"SUBSCRIBE (m23, q1, 'some/topic')")
	writeReadAndCompare(t, NewSubscribe(23, Topic{Name: "some/topic", QoS: 0}, Topic{Name: "some/other"}),
		"SUBSCRIBE (m23, [(q0, 'some/topic'), (q0, 'some/other')])")
}

func TestParseSubscribe_badFlags(t *testing.T) {
	testutils.EnsureFailed(t, func(st *testing.T) {
		Parse(st, mqtt.NewReader(bytes.NewReader([]byte{TpSubscribe | 1, 0})))
	})
}

func TestParseSubAck(t *testing.T) {
	writeReadAndCompare(t, NewSubAck(23, 1), "SUBACK (m23, rc1)")
	writeReadAndCompare(t, NewSubAck(23, 1, 1), "SUBACK (m23, [rc1, rc1])")
}
