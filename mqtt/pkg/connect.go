package pkg

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/tada/mqtt-nats/mqtt"
)

const (
	protoName = "MQTT"

	cleanSessionFlag = byte(0x02)
	willFlag         = byte(0x04)
	willQoS          = byte(0x18)
	willRetainFlag   = byte(0x20)
	passwordFlag     = byte(0x40)
	userNameFlag     = byte(0x80)
)

// ReturnCode used in response to a CONNECT
type ReturnCode byte

func (r ReturnCode) Error() string {
	switch r {
	case RtAccepted:
		return "accepted"
	case RtUnacceptableProtocolVersion:
		return "unacceptable protocol version"
	case RtIdentifierRejected:
		return "identifier rejected"
	case RtServerUnavailable:
		return "server unavailable"
	case RtBadUserNameOrPassword:
		return "bad user name or password"
	case RtNotAuthorized:
		return "not authorized"
	default:
		return "unknown error"
	}
}

const (
	// RtAccepted Connection Accepted
	RtAccepted = ReturnCode(iota)

	// RtUnacceptableProtocolVersion The Server does not support the level of the MQTT protocol requested by the Client
	RtUnacceptableProtocolVersion

	// RtIdentifierRejected The Client identifier is correct UTF-8 but not allowed by the Server
	RtIdentifierRejected

	// RtServerUnavailable The Network Connection has been made but the MQTT service is unavailable
	RtServerUnavailable

	// RtBadUserNameOrPassword The data in the user name or password is malformed
	RtBadUserNameOrPassword

	// RtNotAuthorized The Client is not authorized to connect
	RtNotAuthorized
)

// Will is the optional client will in the MQTT connect packet
type Will struct {
	Topic   string
	Message []byte
	QoS     byte
	Retain  bool
}

// Equals returns true if this instance is equal to the given instance, false if not
func (w *Will) Equals(ow *Will) bool {
	return w.Retain == ow.Retain && w.QoS == ow.QoS && w.Topic == ow.Topic && bytes.Equal(w.Message, ow.Message)
}

// Connect is the MQTT connect packet
type Connect struct {
	clientID    string
	creds       *Credentials
	will        *Will
	keepAlive   uint16
	clientLevel byte
	flags       byte
}

// NewConnect creates a new MQTT connect packet
func NewConnect(clientID string, cleanSession bool, keepAlive uint16, will *Will, creds *Credentials) *Connect {
	flags := byte(0)
	if cleanSession {
		flags |= cleanSessionFlag
	}
	if will != nil {
		flags |= willFlag | (will.QoS << 3)
		if will.Retain {
			flags |= willRetainFlag
		}
	}
	if creds != nil {
		if len(creds.Password) > 0 {
			flags |= passwordFlag
		}
		if len(creds.User) > 0 {
			flags |= userNameFlag
		}
	}

	return &Connect{
		clientID:    clientID,
		creds:       creds,
		keepAlive:   keepAlive,
		clientLevel: 0x4,
		flags:       flags,
		will:        will}
}

// ParseConnect parses the connect packet from the given reader.
func ParseConnect(r *mqtt.Reader, _ byte, pkLen int) (Packet, error) {
	var err error
	if r, err = r.ReadPacket(pkLen); err != nil {
		return nil, err
	}

	// Protocol Name
	var proto string
	if proto, err = r.ReadString(); err != nil {
		return nil, err
	}
	if proto != protoName {
		return nil, fmt.Errorf(`expected connect packet with protocol name "MQTT", got "%s"`, proto)
	}

	c := &Connect{}

	// Protocol Level
	if c.clientLevel, err = r.ReadByte(); err != nil {
		return nil, err
	}
	if c.clientLevel != 0x4 {
		return c, RtUnacceptableProtocolVersion
	}

	// Connect Flags
	if c.flags, err = r.ReadByte(); err != nil {
		return nil, err
	}

	// Keep Alive
	if c.keepAlive, err = r.ReadUint16(); err != nil {
		return nil, err
	}

	// Payload starts here

	// Client Identifier
	if c.clientID, err = r.ReadString(); err != nil {
		return nil, err
	}

	// Will
	if c.HasWill() {
		c.will = &Will{QoS: (c.flags & willQoS) >> 3, Retain: (c.flags & willRetainFlag) != 0}
		if c.will.Topic, err = r.ReadString(); err != nil {
			return nil, err
		}
		if c.will.Message, err = r.ReadBytes(); err != nil {
			return nil, err
		}
	}

	// User Name
	if (c.flags & (userNameFlag | passwordFlag)) != 0 {
		c.creds = &Credentials{}
		if c.HasUserName() {
			if c.creds.User, err = r.ReadString(); err != nil {
				return nil, err
			}
		}
		// Password
		if c.HasPassword() {
			if c.creds.Password, err = r.ReadBytes(); err != nil {
				return nil, err
			}
		}
	}
	return c, nil
}

// ID always returns 0 for a connection packet
func (c *Connect) ID() uint16 {
	return 0
}

// Equals returns true if this packet is equal to the given packet, false if not
func (c *Connect) Equals(p Packet) bool {
	oc, ok := p.(*Connect)
	return ok &&
		c.keepAlive == oc.keepAlive &&
		c.clientLevel == oc.clientLevel &&
		c.flags == oc.flags &&
		c.clientID == oc.clientID &&
		(c.will == oc.will || (c.will != nil && c.will.Equals(oc.will))) &&
		(c.creds == oc.creds || (c.creds != nil && c.creds.Equals(oc.creds)))
}

// SetClientLevel sets the client level. Intended for testing purposes
func (c *Connect) SetClientLevel(cl byte) {
	c.clientLevel = cl
}

// Write writes the MQTT bits of this packet on the given Writer
func (c *Connect) Write(w *mqtt.Writer) {
	pkLen := 2 + len(protoName) +
		1 + // clientLevel
		1 + // flags
		2 + // keepAlive
		2 + len(c.clientID)

	if c.HasWill() {
		pkLen += 2 + len(c.will.Topic)
		pkLen += 2 + len(c.will.Message)
	}
	if c.HasUserName() {
		pkLen += 2 + len(c.creds.User)
	}
	if c.HasPassword() {
		pkLen += 2 + len(c.creds.Password)
	}

	w.WriteU8(TpConnect)
	w.WriteVarInt(pkLen)
	w.WriteString(protoName)
	w.WriteU8(c.clientLevel)
	w.WriteU8(c.flags)
	w.WriteU16(c.keepAlive)
	w.WriteString(c.clientID)
	if c.HasWill() {
		w.WriteString(c.will.Topic)
		w.WriteBytes(c.will.Message)
	}
	if c.HasUserName() {
		w.WriteString(c.creds.User)
	}
	if c.HasPassword() {
		w.WriteBytes(c.creds.Password)
	}
}

// CleanSession returns true if the connection is requesting a clean session
func (c *Connect) CleanSession() bool {
	return (c.flags & cleanSessionFlag) != 0
}

// ClientID returns the id provided by the client
func (c *Connect) ClientID() string {
	return c.clientID
}

// HasPassword returns true if the connection contains a password
func (c *Connect) HasPassword() bool {
	return (c.flags & passwordFlag) != 0
}

// HasUserName returns true if the connection contains a user name
func (c *Connect) HasUserName() bool {
	return (c.flags & userNameFlag) != 0
}

// HasWill returns true if the connection contains a will
func (c *Connect) HasWill() bool {
	return (c.flags & willFlag) != 0
}

// KeepAlive returns the desired keep alive duration
func (c *Connect) KeepAlive() time.Duration {
	return time.Duration(c.keepAlive) * time.Second
}

// Credentials returns the user name and password credentials or nil
func (c *Connect) Credentials() *Credentials {
	return c.creds
}

// String returns a brief string representation of the packet. Suitable for logging
func (c *Connect) String() string {
	return "CONNECT"
}

// Will returns the client will or nil
func (c *Connect) Will() *Will {
	return c.will
}

// DeleteWill clears will and all flags that are associated with the will
func (c *Connect) DeleteWill() {
	c.flags &^= willFlag | willQoS | willRetainFlag
	c.will = nil
}

// AckConnect is the MQTT CONNACK packet sent in response to a CONNECT
type AckConnect struct {
	flags      byte
	returnCode byte
}

// NewAckConnect creates an CONNACK packet
func NewAckConnect(sessionPresent bool, returnCode ReturnCode) Packet {
	flags := byte(0x00)
	if sessionPresent {
		flags |= 0x01
	}
	return &AckConnect{flags: flags, returnCode: byte(returnCode)}
}

// ParseAckConnect parses a CONNACK packet
func ParseAckConnect(r *mqtt.Reader, _ byte, pkLen int) (Packet, error) {
	var err error
	if pkLen != 2 {
		return nil, errors.New("malformed CONNACK")
	}
	bs := make([]byte, 2)
	_, err = r.Read(bs)
	if err != nil {
		return nil, err
	}
	return &AckConnect{flags: bs[0], returnCode: bs[1]}, nil
}

// ID always returns 0 for a CONNACK packet
func (a *AckConnect) ID() uint16 {
	return 0
}

// Equals returns true if this packet is equal to the given packet, false if not
func (a *AckConnect) Equals(p Packet) bool {
	ac, ok := p.(*AckConnect)
	return ok && *a == *ac
}

// String returns a brief string representation of the packet. Suitable for logging
func (a *AckConnect) String() string {
	return fmt.Sprintf("CONNACK (s%d, rt%d)", a.flags, a.returnCode)
}

// Write writes the MQTT bits of this packet on the given Writer
func (a *AckConnect) Write(w *mqtt.Writer) {
	w.WriteU8(TpConnAck)
	w.WriteU8(2)
	w.WriteU8(a.flags)
	w.WriteU8(a.returnCode)
}

// The Disconnect type represents the MQTT DISCONNECT packet
type Disconnect int

// DisconnectSingleton is the one and only instance of the Disconnect type
const DisconnectSingleton = Disconnect(0)

// ID always returns 0 for a DISCONNECT packet
func (a Disconnect) ID() uint16 {
	return 0
}

// Equals returns true if this packet is equal to the given packet, false if not
func (Disconnect) Equals(p Packet) bool {
	return p == DisconnectSingleton
}

// String returns a brief string representation of the packet. Suitable for logging
func (Disconnect) String() string {
	return "DISCONNECT"
}

// Write writes the MQTT bits of this packet on the given Writer
func (Disconnect) Write(w *mqtt.Writer) {
	w.WriteU8(TpDisconnect)
	w.WriteU8(0)
}
