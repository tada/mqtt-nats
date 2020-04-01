// Package pkg contains the MQTT package structures
package pkg

import "github.com/tada/mqtt-nats/mqtt"

const (
	TpConnect     = 0x10
	TpConnAck     = 0x20
	TpPublish     = 0x30
	TpPubAck      = 0x40
	TpPubRec      = 0x50
	TpPubRel      = 0x60
	TpPubComp     = 0x70
	TpSubscribe   = 0x80
	TpSubAck      = 0x90
	TpUnsubscribe = 0xa0
	TpUnsubAck    = 0xb0
	TpPing        = 0xc0
	TpPingResp    = 0xd0
	TpDisconnect  = 0xe0
	TpMask        = 0xf0
)

type Package interface {
	// Equals returns true if the receiver is equal to the given package
	Equals(other Package) bool

	// Write writes this package to the given writer
	Write(w *mqtt.Writer)

	// Type returns the MQTT Package type
	Type() byte
}
