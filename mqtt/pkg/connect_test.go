package pkg

import (
	"testing"

	"github.com/tada/mqtt-nats/mqtt"
)

func TestParseConnect(t *testing.T) {
	c1 := NewConnect(`cid`, true, 5, &Will{
		Topic:   "my/will",
		Message: []byte("the will"),
		QoS:     1,
		Retain:  false,
	}, &Credentials{User: "bob", Password: []byte("password")})
	writeReadAndCompare(t, c1, ParseConnect, "CONNECT (c1, k5, u1, p1, w(r0, q1, 'my/will', ... (8 bytes)))")
}

func TestParseConnAck(t *testing.T) {
	writeReadAndCompare(t, NewAckConnect(false, 1), ParseAckConnect, "CONNACK (s0, rt1)")
}

func TestParseDisconnect(t *testing.T) {
	parse := func(*mqtt.Reader, byte, int) (Packet, error) {
		return DisconnectSingleton, nil
	}
	writeReadAndCompare(t, DisconnectSingleton, parse, "DISCONNECT")
}
