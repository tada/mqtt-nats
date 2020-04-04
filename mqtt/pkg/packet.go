// Package pkg contains the MQTT packet structures
package pkg

import "github.com/tada/mqtt-nats/mqtt"

const (
	// TpConnect is the MQTT CONNECT type
	TpConnect = 0x10

	// TpConnAck is the MQTT CONNACK type
	TpConnAck = 0x20

	// TpPublish is the MQTT PUBLISH type
	TpPublish = 0x30

	// TpPubAck is the MQTT PUBACK type
	TpPubAck = 0x40

	// TpPubRec is the MQTT PUBREC type
	TpPubRec = 0x50

	// TpPubRel is the MQTT PUBREL type
	TpPubRel = 0x60

	// TpPubComp is the MQTT PUBCOMP type
	TpPubComp = 0x70

	// TpSubscribe is the MQTT SUBSCRIBE type
	TpSubscribe = 0x80

	// TpSubAck is the MQTT SUBACK type
	TpSubAck = 0x90

	// TpUnsubscribe is the MQTT UNSUBSCRIBE type
	TpUnsubscribe = 0xa0

	// TpUnsubAck is the MQTT UNSUBACK type
	TpUnsubAck = 0xb0

	// TpPing is the MQTT PINGREQ type
	TpPing = 0xc0

	// TpPingResp is the MQTT PINGRESP type
	TpPingResp = 0xd0

	// TpDisconnect is the MQTT DISCONNECT type
	TpDisconnect = 0xe0

	// TpMask is bitmask for the MQTT type
	TpMask = 0xf0
)

// The Packet interface is implemented by all MQTT packet types
type Packet interface {
	// ID returns the packet ID or 0 if not applicable
	ID() uint16

	// Equals returns true if this packet is equal to the given packet, false if not
	Equals(other Packet) bool

	// Write writes the MQTT bits of this packet on the given Writer
	Write(w *mqtt.Writer)
}
