package pkg

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/tada/mqtt-nats/jsonstream"
	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/pio"
)

const (
	PublishRetain = 0x01
	PublishQoS    = 0x06
	PublishDup    = 0x08
)

type Publish struct {
	name     string
	replyTo  string
	payload  []byte
	id       uint16
	flags    byte
	sentByUs bool // set if the message originated from this server (happens when a client will is published)
}

// SimplePublish creates a new Publish package with all flags zero and no reply
func SimplePublish(topic string, payload []byte) *Publish {
	return &Publish{name: topic, payload: payload}
}

// NewPublish creates a new Publish package
func NewPublish(id uint16, topic string, flags byte, payload []byte, sentByUs bool, natsReplyTo string) *Publish {
	return &Publish{
		name:     topic,
		replyTo:  natsReplyTo,
		payload:  payload,
		id:       id,
		flags:    flags,
		sentByUs: sentByUs,
	}
}

// NewPublish2 creates a new Publish package
func NewPublish2(id uint16, topic string, payload []byte, qos byte, dup bool, retain bool) *Publish {
	flags := byte(0)
	if qos > 0 {
		flags |= qos << 1
	}
	if dup {
		flags |= PublishDup
	}
	if retain {
		flags |= PublishRetain
	}
	return &Publish{id: id, flags: flags, name: topic, payload: payload, sentByUs: false, replyTo: ""}
}

// ParsePublish parses the publish package from the given reader.
func ParsePublish(r *mqtt.Reader, flags byte, pkLen int) (*Publish, error) {
	var err error
	if r, err = r.ReadPackage(pkLen); err != nil {
		return nil, err
	}

	pp := &Publish{flags: flags & 0xf}
	if pp.name, err = r.ReadString(); err != nil {
		return nil, err
	}

	if pp.QoSLevel() > 0 {
		if pp.id, err = r.ReadUint16(); err != nil {
			return nil, err
		}
	}
	if pp.payload, err = r.ReadRemainingBytes(); err != nil {
		return nil, err
	}
	return pp, nil
}

func (p *Publish) Equals(other Package) bool {
	op, ok := other.(*Publish)
	return ok &&
		p.id == op.id &&
		p.flags == op.flags &&
		p.sentByUs == op.sentByUs &&
		p.name == op.name &&
		p.replyTo == op.replyTo &&
		bytes.Equal(p.payload, op.payload)
}

// Flags returns the package flags
func (p *Publish) Flags() byte {
	return p.flags
}

// ID returns the MQTT Package Identifier. The identifier is only valid if QoS > 0
func (p *Publish) ID() uint16 {
	return p.id
}

// IsDup returns true if the package is a duplicate of a previously sent package
func (p *Publish) IsDup() bool {
	return (p.flags & PublishDup) != 0
}

func (p *Publish) SetDup() {
	p.flags |= PublishDup
}

func (p *Publish) MarshalJSON() ([]byte, error) {
	return jsonstream.Marshal(p)
}

func (p *Publish) MarshalToJSON(w io.Writer) {
	pio.WriteString(`{"flags":`, w)
	pio.WriteInt(int64(p.flags), w)
	pio.WriteString(`,"id":`, w)
	pio.WriteInt(int64(p.id), w)
	pio.WriteString(`,"name":`, w)
	jsonstream.WriteString(p.name, w)
	if p.replyTo != "" {
		pio.WriteString(`,"reply_to":`, w)
		jsonstream.WriteString(p.replyTo, w)
	}
	if len(p.payload) > 0 {
		pls := string(p.payload)
		if utf8.ValidString(pls) {
			pio.WriteString(`,"payload":`, w)
			jsonstream.WriteString(pls, w)
		} else {
			pio.WriteString(`,"payload_enc":`, w)
			jsonstream.WriteString(base64.StdEncoding.EncodeToString(p.payload), w)
		}
	}
	pio.WriteByte('}', w)
}

// NatsReplyTo returns the NATS replyTo subject. Only valid when the package represents something
// received from NATS due to a client subscribing to a topic with QoS level > 0
func (p *Publish) NatsReplyTo() string {
	return p.replyTo
}

// Payload retursn the payload of the published message
func (p *Publish) Payload() []byte {
	return p.payload
}

// QoSLevel returns the quality of service level which is 0, 1 or 2.
func (p *Publish) QoSLevel() byte {
	return (p.flags & PublishQoS) >> 1
}

func (p *Publish) ResetRetain() {
	p.flags &^= PublishRetain
}

func (p *Publish) Retain() bool {
	return (p.flags & PublishRetain) != 0
}

func (p *Publish) String() string {
	// layout borrowed from mosquitto_sub log output
	return fmt.Sprintf("PUBLISH (d%d, q%d, r%b, m%d, '%s', ... (%d bytes))",
		(p.flags&0x08)>>3,
		p.QoSLevel(),
		p.flags&0x01,
		p.ID(),
		p.name,
		len(p.payload))
}

// TopicName returns the name of the topic
func (p *Publish) TopicName() string {
	return p.name
}

// Type returns TpPublish
func (p *Publish) Type() byte {
	return TpPublish
}

func (p *Publish) UnmarshalJSON(bs []byte) error {
	return jsonstream.Unmarshal(p, bs)
}

func (p *Publish) UnmarshalFromJSON(js *json.Decoder, t json.Token) {
	jsonstream.AssertDelimToken(t, '{')
	for {
		s, ok := jsonstream.AssertStringOrEnd(js, '}')
		if !ok {
			break
		}
		switch s {
		case "flags":
			p.flags = byte(jsonstream.AssertInt(js))
		case "id":
			p.id = uint16(jsonstream.AssertInt(js))
		case "name":
			p.name = jsonstream.AssertString(js)
		case "replyTo":
			p.replyTo = jsonstream.AssertString(js)
		case "payload":
			p.payload = []byte(jsonstream.AssertString(js))
		case "payload_enc":
			var err error
			p.payload, err = base64.StdEncoding.DecodeString(jsonstream.AssertString(js))
			if err != nil {
				panic(pio.Error{Cause: err})
			}
		}
	}
}

// Write emits the package using the given Writer
func (p *Publish) Write(w *mqtt.Writer) {
	w.WriteU8(TpPublish | p.flags)
	pkLen := 2 + len(p.name) + len(p.payload)
	if p.QoSLevel() > 0 {
		pkLen += 2
	}
	w.WriteVarInt(pkLen)
	w.WriteString(p.name)
	if p.QoSLevel() > 0 {
		w.WriteU16(p.id)
	}
	_, _ = w.Write(p.payload)
}

type PubAck uint16

func ParsePubAck(r *mqtt.Reader, _ byte, pkLen int) (PubAck, error) {
	if pkLen != 2 {
		return 0, errors.New("malformed PUBACK")
	}
	id, err := r.ReadUint16()
	return PubAck(id), err
}

func (p PubAck) Equals(other Package) bool {
	return p == other
}

func (p PubAck) ID() uint16 {
	return uint16(p)
}

func (p PubAck) String() string {
	return fmt.Sprintf("PUBACK (m%d)", int(p))
}

func (p PubAck) Type() byte {
	return TpPubAck
}

func (p PubAck) Write(w *mqtt.Writer) {
	w.WriteU8(TpPubAck)
	w.WriteU8(2)
	w.WriteU16(uint16(p))
}

type PubRec uint16

func ParsePubRec(r *mqtt.Reader, _ byte, pkLen int) (PubRec, error) {
	if pkLen != 2 {
		return 0, errors.New("malformed PUBREC")
	}
	id, err := r.ReadUint16()
	return PubRec(id), err
}

func (p PubRec) Equals(other Package) bool {
	return p == other
}

func (p PubRec) ID() uint16 {
	return uint16(p)
}

func (p PubRec) String() string {
	return fmt.Sprintf("PUBREC (m%d)", int(p))
}

func (p PubRec) Type() byte {
	return TpPubRec
}

func (p PubRec) Write(w *mqtt.Writer) {
	w.WriteU8(TpPubRec)
	w.WriteU8(2)
	w.WriteU16(uint16(p))
}

type PubRel uint16

func ParsePubRel(r *mqtt.Reader, _ byte, pkLen int) (PubRel, error) {
	if pkLen != 2 {
		return 0, errors.New("malformed PUBREL")
	}
	id, err := r.ReadUint16()
	return PubRel(id), err
}

func (p PubRel) Equals(other Package) bool {
	return p == other
}

func (p PubRel) ID() uint16 {
	return uint16(p)
}

func (p PubRel) String() string {
	return fmt.Sprintf("PUBREL (m%d)", int(p))
}

func (p PubRel) Type() byte {
	return TpPubRel
}

func (p PubRel) Write(w *mqtt.Writer) {
	w.WriteU8(TpPubRel)
	w.WriteU8(2)
	w.WriteU16(uint16(p))
}

type PubComp uint16

func ParsePubComp(r *mqtt.Reader, _ byte, pkLen int) (PubComp, error) {
	if pkLen != 2 {
		return 0, errors.New("malformed PUBCOMP")
	}
	id, err := r.ReadUint16()
	return PubComp(id), err
}

func (p PubComp) Equals(other Package) bool {
	return p == other
}

func (p PubComp) ID() uint16 {
	return uint16(p)
}

func (p PubComp) String() string {
	return fmt.Sprintf("PUBCOMP (m%d)", int(p))
}

func (p PubComp) Type() byte {
	return TpPubComp
}

func (p PubComp) Write(w *mqtt.Writer) {
	w.WriteU8(TpPubComp)
	w.WriteU8(2)
	w.WriteU16(uint16(p))
}
