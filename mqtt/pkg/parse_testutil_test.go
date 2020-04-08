package pkg

import (
	"bytes"
	"testing"

	"github.com/tada/mqtt-nats/testutils"
)

func TestParse_unknownPacket(t *testing.T) {
	testutils.EnsureFailed(t, func(st *testing.T) {
		Parse(st, bytes.NewReader([]byte{0xf0, 0}))
	}, "unknown packet type 15")
}
