package pkg

import (
	"testing"
)

func TestParseConnect(t *testing.T) {
	c1 := NewConnect(`cid`, true, 5, &Will{
		Topic:   "my/will",
		Message: []byte("the will"),
		QoS:     1,
		Retain:  false,
	}, &Credentials{User: "bob", Password: []byte("password")})
	writeReadAndCompare(t, c1, "CONNECT (c1, k5, u1, p1, w(r0, q1, 'my/will', ... (8 bytes)))")
}

func TestParseConnAck(t *testing.T) {
	writeReadAndCompare(t, NewConnAck(false, 1), "CONNACK (s0, rt1)")
}

func TestParseDisconnect(t *testing.T) {
	writeReadAndCompare(t, DisconnectSingleton, "DISCONNECT")
}
