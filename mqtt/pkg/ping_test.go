package pkg

import (
	"testing"
)

func TestParsePingReq(t *testing.T) {
	writeReadAndCompare(t, PingRequestSingleton, "PINGREQ")
}

func TestParsePingResp(t *testing.T) {
	writeReadAndCompare(t, PingResponseSingleton, "PINGRESP")
}
