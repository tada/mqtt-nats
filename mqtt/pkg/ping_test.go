package pkg

import (
	"testing"

	"github.com/tada/mqtt-nats/mqtt"
)

func TestParsePingReq(t *testing.T) {
	writeReadAndCompare(t, PingRequestSingleton, func(*mqtt.Reader, byte, int) (Packet, error) {
		return PingRequestSingleton, nil
	}, "PINGREQ")
}

func TestParsePingResp(t *testing.T) {
	writeReadAndCompare(t, PingResponseSingleton, func(*mqtt.Reader, byte, int) (Packet, error) {
		return PingResponseSingleton, nil
	}, "PINGRESP")
}
