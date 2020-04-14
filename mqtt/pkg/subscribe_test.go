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

func TestParseSubscribe_badLen(t *testing.T) {
	_, err := pkg.ParseSubscribe(mqtt.NewReader(bytes.NewReader([]byte{})), pkg.TpSubscribe|2, 8)
	utils.CheckError(err, t)
}

func TestParseSubscribe_badId(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteU16(28)
	_, err := pkg.ParseSubscribe(mqtt.NewReader(bytes.NewReader([]byte{1})), pkg.TpConnAck|2, 1)
	utils.CheckError(err, t)
}

func TestParseSubscribe_badCount(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteU16(28)
	_, err := pkg.ParseSubscribe(mqtt.NewReader(bytes.NewReader([]byte{0, 2, 3})), pkg.TpConnAck|2, 3)
	utils.CheckError(err, t)
}

func TestParseSubscribe_badTopic(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteU16(28)
	w.WriteU16(1)
	_, err := pkg.ParseSubscribe(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck|2, 4)
	utils.CheckError(err, t)
}

func TestParseSubscribe_badQoS(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteU16(28)
	w.WriteString("tpc")
	_, err := pkg.ParseSubscribe(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck|2, 7)
	utils.CheckError(err, t)
}

func TestParseSubscribe_invalidQoS(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteU16(28)
	w.WriteString("tpc")
	w.WriteU8(3)
	_, err := pkg.ParseSubscribe(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck|2, 8)
	utils.CheckError(err, t)
}

func TestSubscribe_Equals(t *testing.T) {
	a := pkg.NewSubscribe(32, pkg.Topic{Name: "a"}, pkg.Topic{Name: "b", QoS: 1})
	b := pkg.NewSubscribe(32, pkg.Topic{Name: "a"}, pkg.Topic{Name: "b", QoS: 1})
	utils.CheckTrue(a.Equals(b), t)

	b = pkg.NewSubscribe(32, pkg.Topic{Name: "a"}, pkg.Topic{Name: "b", QoS: 2})
	utils.CheckFalse(a.Equals(b), t)

	c := pkg.NewSubAck(32, 0, 2)
	utils.CheckFalse(a.Equals(c), t)
}

func TestParseSubAck(t *testing.T) {
	writeReadAndCompare(t, pkg.NewSubAck(23, 1), "SUBACK (m23, rc1)")
	writeReadAndCompare(t, pkg.NewSubAck(23, 1, 1), "SUBACK (m23, [rc1, rc1])")
}

func TestSubAck_Equals(t *testing.T) {
	a := pkg.NewSubAck(32, 1, 2, 0)
	b := pkg.NewSubAck(32, 1, 2, 0)
	utils.CheckTrue(a.Equals(b), t)

	b = pkg.NewSubAck(32, 1, 2, 1)
	utils.CheckFalse(a.Equals(b), t)

	c := pkg.NewSubscribe(32, pkg.Topic{Name: "a"}, pkg.Topic{Name: "b", QoS: 1})
	utils.CheckFalse(a.Equals(c), t)
}
