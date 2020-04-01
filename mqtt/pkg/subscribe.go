package pkg

import (
	"bytes"
	"errors"
	"strconv"

	"github.com/tada/mqtt-nats/mqtt"
)

// Topic is an MQTT Topic subscription with name and desired quality of service
type Topic struct {
	// Name is the Topic Name
	Name string

	// QoS Quality of Service, will be 0, 1, or 2.
	QoS byte
}

// Subscribe is the MQTT subscribe package
type Subscribe struct {
	id     uint16
	topics []Topic
}

const FixedSubscribeFlags = 2

// NewSubscribe creates a new MQTT subscribe package
func NewSubscribe(id uint16, topics ...Topic) *Subscribe {
	return &Subscribe{id: id, topics: topics}
}

// ParseSubscribe parses the subscribe package from the given reader.
func ParseSubscribe(r *mqtt.Reader, b byte, pkLen int) (*Subscribe, error) {
	if (b & 0xf) != FixedSubscribeFlags {
		return nil, errors.New("malformed subscribe header")
	}

	var err error
	if r, err = r.ReadPackage(pkLen); err != nil {
		return nil, err
	}

	sp := &Subscribe{}
	if sp.id, err = r.ReadUint16(); err != nil {
		return nil, err
	}

	for r.Len() > 0 {
		t := Topic{}
		if t.Name, err = r.ReadString(); err != nil {
			return nil, err
		}
		if t.QoS, err = r.ReadByte(); err != nil {
			return nil, err
		}
		if t.QoS > 2 {
			return nil, errors.New("malformed subscribed topic QoS")
		}
		sp.topics = append(sp.topics, t)
	}
	return sp, nil
}

// ID returns the MQTT Package Identifier
func (s *Subscribe) ID() uint16 {
	return s.id
}

func (s *Subscribe) Equals(p Package) bool {
	if os, ok := p.(*Subscribe); ok && s.id == os.id && len(s.topics) == len(os.topics) {
		for i := range s.topics {
			if s.topics[i] != os.topics[i] {
				return false
			}
		}
		return true
	}
	return false
}

func (s *Subscribe) String() string {
	bs := bytes.NewBufferString("SUBSCRIBE (m")
	bs.WriteString(strconv.Itoa(int(s.ID())))
	bs.WriteString(", ")
	wt := func(t Topic) {
		bs.WriteByte('q')
		bs.WriteString(strconv.Itoa(int(t.QoS)))
		bs.WriteString(", '")
		bs.WriteString(t.Name)
		bs.WriteByte('\'')
	}
	if len(s.topics) != 1 {
		bs.WriteByte('[')
		for i, t := range s.topics {
			if i > 0 {
				bs.WriteString(", ")
			}
			bs.WriteByte('(')
			wt(t)
			bs.WriteByte(')')
		}
		bs.WriteByte(']')
	} else {
		wt(s.topics[0])
	}
	bs.WriteByte(')')
	return bs.String()
}

// Topics returns the list of topics to subscribe to
func (s *Subscribe) Topics() []Topic {
	return s.topics
}

// Type returns the TpSubscribe package type
func (s *Subscribe) Type() byte {
	return TpSubscribe
}

func (s *Subscribe) Write(w *mqtt.Writer) {
	pkLen := 2 // id
	for i := range s.topics {
		pkLen += 3 + len(s.topics[i].Name)
	}
	w.WriteU8(TpSubscribe | FixedSubscribeFlags)
	w.WriteVarInt(pkLen)
	w.WriteU16(s.id)
	for i := range s.topics {
		t := s.topics[i]
		w.WriteString(t.Name)
		w.WriteU8(t.QoS)
	}
}

type SubAck struct {
	id           uint16
	topicReturns []byte
}

func NewSubAck(id uint16, topicReturns ...byte) *SubAck {
	return &SubAck{id: id, topicReturns: topicReturns}
}

func ParseSubAck(r *mqtt.Reader, b byte, pkLen int) (*SubAck, error) {
	var err error
	if r, err = r.ReadPackage(pkLen); err != nil {
		return nil, err
	}
	s := &SubAck{}
	if s.id, err = r.ReadUint16(); err != nil {
		return nil, err
	}
	if s.topicReturns, err = r.ReadExact(pkLen - 2); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SubAck) Equals(p Package) bool {
	os, ok := p.(*SubAck)
	return ok && s.id == os.id && bytes.Equal(s.topicReturns, os.topicReturns)
}

func (s *SubAck) ID() uint16 {
	return s.id
}

func (s *SubAck) String() string {
	bs := bytes.NewBufferString("SUBACK (m")
	bs.WriteString(strconv.Itoa(int(s.id)))
	bs.WriteString(", ")
	if len(s.topicReturns) != 1 {
		bs.WriteByte('[')
		for i, t := range s.topicReturns {
			if i > 0 {
				bs.WriteString(", ")
			}
			bs.WriteString("rc")
			bs.WriteString(strconv.Itoa(int(t)))
		}
		bs.WriteByte(']')
	} else {
		bs.WriteString("rc")
		bs.WriteString(strconv.Itoa(int(s.topicReturns[0])))
	}
	bs.WriteByte(')')
	return bs.String()
}

func (s *SubAck) TopicReturns() []byte {
	return s.topicReturns
}

func (s *SubAck) Type() byte {
	return TpSubAck
}

func (s *SubAck) Write(w *mqtt.Writer) {
	w.WriteU8(TpSubAck)
	w.WriteVarInt(2 + len(s.topicReturns))
	w.WriteU16(s.id)
	_, _ = w.Write(s.topicReturns)
}
