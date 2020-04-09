// +build citest

package packet

import (
	"fmt"
	"io"
	"testing"

	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/mqtt/pkg"
)

// Parse is a test helper function that parses the next package from the given reader and returns it. t.Fatal(err)
// will be called if an error occurs when reading or parsing.
func Parse(t *testing.T, rdr io.Reader) pkg.Packet {
	t.Helper()
	// Read packet type and flags
	r := mqtt.NewReader(rdr)
	var p pkg.Packet
	b, err := r.ReadByte()
	if err == nil {
		pkgType := b & pkg.TpMask

		// Read packet length
		var rl int
		rl, err = r.ReadVarInt()
		if err == nil {
			switch pkgType {
			case pkg.TpConnAck:
				p, err = pkg.ParseConnAck(r, b, rl)
			case pkg.TpDisconnect:
				p = pkg.DisconnectSingleton
			case pkg.TpPing:
				p = pkg.PingRequestSingleton
			case pkg.TpPingResp:
				p = pkg.PingResponseSingleton
			case pkg.TpConnect:
				p, err = pkg.ParseConnect(r, b, rl)
			case pkg.TpPublish:
				p, err = pkg.ParsePublish(r, b, rl)
			case pkg.TpPubAck:
				p, err = pkg.ParsePubAck(r, b, rl)
			case pkg.TpPubRec:
				p, err = pkg.ParsePubRec(r, b, rl)
			case pkg.TpPubRel:
				p, err = pkg.ParsePubRel(r, b, rl)
			case pkg.TpPubComp:
				p, err = pkg.ParsePubComp(r, b, rl)
			case pkg.TpSubscribe:
				p, err = pkg.ParseSubscribe(r, b, rl)
			case pkg.TpSubAck:
				p, err = pkg.ParseSubAck(r, b, rl)
			case pkg.TpUnsubscribe:
				p, err = pkg.ParseUnsubscribe(r, b, rl)
			case pkg.TpUnsubAck:
				p, err = pkg.ParseUnsubAck(r, b, rl)
			default:
				err = fmt.Errorf("received unknown packet type %d", (b&pkg.TpMask)>>4)
			}
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	return p
}
