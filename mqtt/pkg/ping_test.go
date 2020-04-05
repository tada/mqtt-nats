package pkg

import (
	"testing"

	"github.com/tada/mqtt-nats/mqtt"
)

func TestParsePingReq(t *testing.T) {
	parse := func(*mqtt.Reader, byte, int) (Packet, error) {
		return PingRequestSingleton, nil
	}
	writeReadAndCompare(t, PingRequestSingleton, parse, "PINGREQ")
}

func TestParsePingResp(t *testing.T) {
	parse := func(*mqtt.Reader, byte, int) (Packet, error) {
		return PingResponseSingleton, nil
	}
	writeReadAndCompare(t, PingResponseSingleton, parse, "PINGRESP")
}
