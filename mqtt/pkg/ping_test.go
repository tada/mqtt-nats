package pkg_test

import (
	"testing"

	"github.com/tada/mqtt-nats/mqtt/pkg"
)

func TestParsePingReq(t *testing.T) {
	writeReadAndCompare(t, pkg.PingRequestSingleton, "PINGREQ")
}

func TestParsePingResp(t *testing.T) {
	writeReadAndCompare(t, pkg.PingResponseSingleton, "PINGRESP")
}
