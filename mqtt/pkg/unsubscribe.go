package pkg

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/tada/mqtt-nats/mqtt"
)

// Unsubscribe is the MQTT UNSUBSCRIBE package
type Unsubscribe struct {
	id     uint16
	topics []string
}

// NewUnsubscribe creates a new Unsubscribe package
func NewUnsubscribe(id uint16, topics ...string) *Unsubscribe {
	return &Unsubscribe{id: id, topics: topics}
}

// ParseUnsubscribe parses the unsubscribe package from the given reader.
func ParseUnsubscribe(r *mqtt.Reader, b byte, pkLen int) (Package, error) {
	if (b & 0xf) != 2 {
		return nil, errors.New("malformed unsubscribe header")
	}

	var err error
	if r, err = r.ReadPackage(pkLen); err != nil {
		return nil, err
	}

	up := &Unsubscribe{}
	if up.id, err = r.ReadUint16(); err != nil {
		return nil, err
	}

	for r.Len() > 0 {
		var name string
		if name, err = r.ReadString(); err != nil {
			return nil, err
		}
		up.topics = append(up.topics, name)
	}
	return up, nil
}

// Write writes the MQTT bits of this package on the given Writer
func (u *Unsubscribe) Write(w *mqtt.Writer) {
	pkLen := 2 // package id
	tps := u.topics
	for i := range tps {
		pkLen += 2 + len(tps[i])
	}
	w.WriteU8(TpUnsubscribe | 2)
	w.WriteVarInt(pkLen)
	w.WriteU16(u.id)
	for i := range tps {
		w.WriteString(tps[i])
	}
}

// Equals returns true if this package is equal to the given package, false if not
func (u *Unsubscribe) Equals(p Package) bool {
	if os, ok := p.(*Unsubscribe); ok && u.id == os.id && len(u.topics) == len(os.topics) {
		for i := range u.topics {
			if u.topics[i] != os.topics[i] {
				return false
			}
		}
		return true
	}
	return false
}

// ID returns the MQTT Package Identifier
func (u *Unsubscribe) ID() uint16 {
	return u.id
}

// String returns a brief string representation of the package. Suitable for logging
func (u *Unsubscribe) String() string {
	bs := bytes.NewBufferString("UNSUBSCRIBE (m")
	bs.WriteString(strconv.Itoa(int(u.ID())))
	bs.WriteString(", [")
	for i, t := range u.topics {
		if i > 0 {
			bs.WriteString(", ")
		}
		bs.WriteByte('\'')
		bs.WriteString(t)
		bs.WriteByte('\'')
	}
	bs.WriteString("])")
	return bs.String()
}

// Topics returns the list of topics to subscribe to
func (u *Unsubscribe) Topics() []string {
	return u.topics
}

// UnsubAck is the MQTT UNSUBACK package
type UnsubAck uint16

// ParseUnsubAck parses the unsubscribe package from the given reader.
func ParseUnsubAck(r *mqtt.Reader, b byte, pkLen int) (Package, error) {
	if pkLen != 2 {
		return UnsubAck(0), errors.New("malformed UNSUBACK")
	}
	id, err := r.ReadUint16()
	return UnsubAck(id), err
}

// ID returns the package ID
func (u UnsubAck) ID() uint16 {
	return uint16(u)
}

// Equals returns true if this package is equal to the given package, false if not
func (u UnsubAck) Equals(other Package) bool {
	return u == other
}

// String returns a brief string representation of the package. Suitable for logging
func (u UnsubAck) String() string {
	return fmt.Sprintf("UNSUBACK (m%d)", int(u))
}

// Write writes the MQTT bits of this package on the given Writer
func (u UnsubAck) Write(w *mqtt.Writer) {
	w.WriteU8(TpUnsubAck)
	w.WriteU8(2)
	w.WriteU16(uint16(u))
}
