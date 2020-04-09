package pkg_test

import (
	"bytes"
	"testing"

	"github.com/tada/mqtt-nats/test/packet"

	"github.com/tada/mqtt-nats/mqtt/pkg"

	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/test/utils"
)

func TestParseSubscribe(t *testing.T) {
	writeReadAndCompare(t, pkg.NewSubscribe(23, pkg.Topic{Name: "some/topic", QoS: 1}),
		"SUBSCRIBE (m23, q1, 'some/topic')")
	writeReadAndCompare(t, pkg.NewSubscribe(23, pkg.Topic{Name: "some/topic", QoS: 0}, pkg.Topic{Name: "some/other"}),
		"SUBSCRIBE (m23, [(q0, 'some/topic'), (q0, 'some/other')])")
}

func TestParseSubscribe_badFlags(t *testing.T) {
	utils.EnsureFailed(t, func(st *testing.T) {
		packet.Parse(st, mqtt.NewReader(bytes.NewReader([]byte{pkg.TpSubscribe | 1, 0})))
	})
}

func TestParseSubAck(t *testing.T) {
	writeReadAndCompare(t, pkg.NewSubAck(23, 1), "SUBACK (m23, rc1)")
	writeReadAndCompare(t, pkg.NewSubAck(23, 1, 1), "SUBACK (m23, [rc1, rc1])")
}
