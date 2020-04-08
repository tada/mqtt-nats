// +build citest

package pkg

import (
	"fmt"
	"io"
	"testing"

	"github.com/tada/mqtt-nats/mqtt"
)

// Parse is a test helper function that parses the next package from the given reader and returns it. t.Fatal(err)
// will be called if an error occurs when reading or parsing.
func Parse(t *testing.T, rdr io.Reader) Packet {
	t.Helper()
	// Read packet type and flags
	r := mqtt.NewReader(rdr)
	var p Packet
	b, err := r.ReadByte()
	if err == nil {
		pkgType := b & TpMask

		// Read packet length
		var rl int
		rl, err = r.ReadVarInt()
		if err == nil {
			switch pkgType {
			case TpConnAck:
				p, err = ParseConnAck(r, b, rl)
			case TpDisconnect:
				p = DisconnectSingleton
			case TpPing:
				p = PingRequestSingleton
			case TpPingResp:
				p = PingResponseSingleton
			case TpConnect:
				p, err = ParseConnect(r, b, rl)
			case TpPublish:
				p, err = ParsePublish(r, b, rl)
			case TpPubAck:
				p, err = ParsePubAck(r, b, rl)
			case TpPubRec:
				p, err = ParsePubRec(r, b, rl)
			case TpPubRel:
				p, err = ParsePubRel(r, b, rl)
			case TpPubComp:
				p, err = ParsePubComp(r, b, rl)
			case TpSubscribe:
				p, err = ParseSubscribe(r, b, rl)
			case TpSubAck:
				p, err = ParseSubAck(r, b, rl)
			case TpUnsubscribe:
				p, err = ParseUnsubscribe(r, b, rl)
			case TpUnsubAck:
				p, err = ParseUnsubAck(r, b, rl)
			default:
				err = fmt.Errorf("received unknown packet type %d", (b&TpMask)>>4)
			}
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	return p
}
