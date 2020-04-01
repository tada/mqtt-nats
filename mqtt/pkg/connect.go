package pkg

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tada/mqtt-nats/mqtt"
)

const (
	protoName = "MQTT"

	cleanSessionFlag = byte(0b00000010)
	willFlag         = byte(0b00000100)
	willQoS          = byte(0b00011000)
	willRetainFlag   = byte(0b00100000)
	passwordFlag     = byte(0b01000000)
	userNameFlag     = byte(0b10000000)
)

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

type Connect struct {
	clientID    string
	willTopic   string
	userName    string
	willMessage []byte
	password    []byte
	keepAlive   uint16
	clientLevel byte
	flags       byte
}

type Will struct {
	Topic   string
	Message []byte
	QoS     byte
	Retain  bool
}

func NewConnect(clientID string, cleanSession bool, keepAlive uint16, will *Will, userName string, password []byte) *Connect {
	flags := byte(0)
	if cleanSession {
		flags |= cleanSessionFlag
	}
	var (
		willTopic   string
		willMessage []byte
	)
	if will != nil {
		flags |= willFlag | (will.QoS << 3)
		if will.Retain {
			flags |= willRetainFlag
		}
		willTopic = will.Topic
		willMessage = will.Message
	}
	if len(password) > 0 {
		flags |= passwordFlag
	}
	if len(userName) > 0 {
		flags |= userNameFlag
	}

	return &Connect{
		clientID:    clientID,
		userName:    userName,
		password:    password,
		keepAlive:   keepAlive,
		clientLevel: 0x4,
		flags:       flags,
		willTopic:   willTopic,
		willMessage: willMessage,
	}
}

// ParseConnect parses the connect package from the given reader.
func ParseConnect(r *mqtt.Reader, _ byte, pkLen int) (*Connect, error) {
	var err error
	if r, err = r.ReadPackage(pkLen); err != nil {
		return nil, err
	}

	// Protocol Name
	var proto string
	if proto, err = r.ReadString(); err != nil {
		return nil, err
	}
	if proto != protoName {
		return nil, fmt.Errorf(`expected connect package with protocol name "MQTT", got "%s"`, proto)
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
		if c.willTopic, err = r.ReadString(); err != nil {
			return nil, err
		}
		if c.willMessage, err = r.ReadBytes(); err != nil {
			return nil, err
		}
	}

	// User Name
	if c.HasUserName() {
		if c.userName, err = r.ReadString(); err != nil {
			return nil, err
		}
	}

	// Password
	if c.HasPassword() {
		if c.password, err = r.ReadBytes(); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *Connect) Equals(p Package) bool {
	oc, ok := p.(*Connect)
	return ok &&
		c.keepAlive == oc.keepAlive &&
		c.clientLevel == oc.clientLevel &&
		c.flags == oc.flags &&
		c.clientID == oc.clientID &&
		c.willTopic == oc.willTopic &&
		c.userName == oc.userName &&
		bytes.Equal(c.willMessage, oc.willMessage) &&
		bytes.Equal(c.password, oc.password)
}

// SetClientLevel sets the client level. Intended for testing purposes
func (c *Connect) SetClientLevel(cl byte) {
	c.clientLevel = cl
}

func (c *Connect) Write(w *mqtt.Writer) {
	pkLen := 2 + len(protoName) +
		1 + // clientLevel
		1 + // flags
		2 + // keepAlive
		2 + len(c.clientID)

	if c.HasWill() {
		pkLen += 2 + len(c.willTopic)
		pkLen += 2 + len(c.willMessage)
	}
	if c.HasUserName() {
		pkLen += 2 + len(c.userName)
	}
	if c.HasPassword() {
		pkLen += 2 + len(c.password)
	}

	w.WriteU8(TpConnect)
	w.WriteVarInt(pkLen)
	w.WriteString(protoName)
	w.WriteU8(c.clientLevel)
	w.WriteU8(c.flags)
	w.WriteU16(c.keepAlive)
	w.WriteString(c.clientID)
	if c.HasWill() {
		w.WriteString(c.willTopic)
		w.WriteBytes(c.willMessage)
	}
	if c.HasUserName() {
		w.WriteString(c.userName)
	}
	if c.HasPassword() {
		w.WriteBytes(c.password)
	}
}

func (c *Connect) CleanSession() bool {
	return (c.flags & cleanSessionFlag) != 0
}

func (c *Connect) ClientID() string {
	return c.clientID
}

func (c *Connect) HasPassword() bool {
	return (c.flags & passwordFlag) != 0
}

func (c *Connect) HasUserName() bool {
	return (c.flags & userNameFlag) != 0
}

func (c *Connect) HasWill() bool {
	return (c.flags & willFlag) != 0
}

func (c *Connect) KeepAlive() uint16 {
	return c.keepAlive
}

func (c *Connect) Password() []byte {
	return c.password
}

func (c *Connect) String() string {
	return "CONNECT"
}

func (c *Connect) Type() byte {
	return TpConnect
}

func (c *Connect) Username() string {
	return c.userName
}

func (c *Connect) Will() *Will {
	if c.HasWill() {
		return &Will{
			Message: c.willMessage,
			Topic:   c.willTopic,
			QoS:     (c.flags & willQoS) >> 3,
			Retain:  (c.flags & willRetainFlag) != 0}
	}
	return nil
}

func (c *Connect) DeleteWill() {
	c.flags &^= (willFlag | willQoS | willRetainFlag)
	c.willTopic = ``
	c.willMessage = nil
}

type AckConnect struct {
	flags      byte
	returnCode byte
}

func NewAckConnect(sessionPresent bool, returnCode ReturnCode) Package {
	flags := byte(0x00)
	if sessionPresent {
		flags |= 0x01
	}
	return &AckConnect{flags: flags, returnCode: byte(returnCode)}
}

func ParseAckConnect(r *mqtt.Reader, _ byte, pkLen int) (*AckConnect, error) {
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

func (a *AckConnect) Equals(p Package) bool {
	ac, ok := p.(*AckConnect)
	return ok && *a == *ac
}

func (a *AckConnect) String() string {
	return fmt.Sprintf("CONNACK (s%d, rt%d)", a.flags, a.returnCode)
}

func (a *AckConnect) Type() byte {
	return TpPingResp
}

func (a *AckConnect) Write(w *mqtt.Writer) {
	w.WriteU8(TpConnAck)
	w.WriteU8(2)
	w.WriteU8(a.flags)
	w.WriteU8(a.returnCode)
}

type Disconnect int

const DisconnectSingleton = Disconnect(0)

func (Disconnect) Equals(p Package) bool {
	return p == DisconnectSingleton
}

func (Disconnect) Type() byte {
	return TpDisconnect
}

func (Disconnect) String() string {
	return "DISCONNECT"
}

func (Disconnect) Write(w *mqtt.Writer) {
	w.WriteU8(TpDisconnect)
	w.WriteU8(0)
}
