package packet

import (
	"bytes"
	"testing"

	"github.com/tada/mqtt-nats/test/utils"
)

func TestParse_unknownPacket(t *testing.T) {
	utils.EnsureFailed(t, func(st *testing.T) {
		Parse(st, bytes.NewReader([]byte{0xf0, 0}))
	}, "unknown packet type 15")
}
