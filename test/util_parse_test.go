package test

import (
	"fmt"
	"io"
	"testing"

	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/mqtt/pkg"
)

func parsePackage(t *testing.T, rdr io.Reader) pkg.Package {
	t.Helper()
	// Read package type and flags
	r := mqtt.NewReader(rdr)
	var (
		b   byte
		rl  int
		err error
		p   pkg.Package
	)
	if b, err = r.ReadByte(); err != nil {
		t.Fatal(err)
	}

	pkgType := b & pkg.TpMask

	// Read package length
	if rl, err = r.ReadVarInt(); err != nil {
		t.Fatal(err)
	}

	switch pkgType {
	case pkg.TpConnAck:
		if p, err = pkg.ParseAckConnect(r, b, rl); err == nil {
			return p
		}
	case pkg.TpDisconnect:
		return pkg.DisconnectSingleton
	case pkg.TpPing:
		return pkg.PingRequestSingleton
	case pkg.TpPingResp:
		return pkg.PingResponseSingleton
	case pkg.TpConnect:
		if p, err = pkg.ParseConnect(r, b, rl); err == nil {
			return p
		}
	case pkg.TpPublish:
		if p, err = pkg.ParsePublish(r, b, rl); err == nil {
			return p
		}
	case pkg.TpPubAck:
		if p, err = pkg.ParsePubAck(r, b, rl); err == nil {
			return p
		}
	case pkg.TpPubRec:
		if p, err = pkg.ParsePubRec(r, b, rl); err == nil {
			return p
		}
	case pkg.TpPubRel:
		if p, err = pkg.ParsePubRel(r, b, rl); err == nil {
			return p
		}
	case pkg.TpPubComp:
		if p, err = pkg.ParsePubComp(r, b, rl); err == nil {
			return p
		}
	case pkg.TpSubscribe:
		if p, err = pkg.ParseSubscribe(r, b, rl); err == nil {
			return p
		}
	case pkg.TpSubAck:
		if p, err = pkg.ParseSubAck(r, b, rl); err == nil {
			return p
		}
	case pkg.TpUnsubscribe:
		if p, err = pkg.ParseUnsubscribe(r, b, rl); err == nil {
			return p
		}
	case pkg.TpUnsubAck:
		if p, err = pkg.ParseUnsubAck(r, b, rl); err == nil {
			return p
		}
	default:
		err = fmt.Errorf("received unknown package type %d", (b&pkg.TpMask)>>4)
	}
	t.Fatal(err)
	return nil
}
