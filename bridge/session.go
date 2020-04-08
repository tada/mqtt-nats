package bridge

import (
	"encoding/json"
	"io"
	"strconv"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/tada/mqtt-nats/jsonstream"
	"github.com/tada/mqtt-nats/mqtt/pkg"
	"github.com/tada/mqtt-nats/pio"
)

// A Session contains data associated with a client ID. The session might survive client
// connections.
type Session interface {
	jsonstream.Consumer
	jsonstream.Streamer

	// ID returns an identifier that is unique for this session
	ID() string

	// ClientID returns the id of the client that this session belongs to
	ClientID() string

	// Destroy the session
	Destroy()

	// AckRequested remembers the given subscription which represents an awaited ACK
	// for the given packetID
	AckRequested(uint16, *nats.Subscription)

	// AwaitsAck returns true if a subscription associated with the given packet identifier
	// is currently waiting for an Ack.
	AwaitsAck(uint16) bool

	// AckReceived will close a pending ack subscription from the session. It returns whether or not
	// such an ack was pending
	AckReceived(uint16) bool

	// ClientAckRequested remembers the id of a packet which has been sent to the client. The packet stems from a NATS
	// subscription with QoS level > 0 and it is now expected that the client sends an PubACK back to which can be
	// propagated to the reply-to address.
	ClientAckRequested(*pkg.Publish)

	// ClientAckReceived will close a pending response ack subscription and forward the ACK to the
	// replyTo subject. It returns whether or not such an ack was pending
	ClientAckReceived(uint16, *nats.Conn) bool

	// Resend all messages that the client hasn't acknowledged
	ResendClientUnack(c *client)

	// RestoreAckSubscriptions called when a client restores an old session. THe method restores subscriptions that
	// were peristed and then loaded again.
	RestoreAckSubscriptions(c *client)
}

type session struct {
	id              string
	clientID        string
	prelAwaitsAck   map[uint16]string
	awaitsAck       map[uint16]*nats.Subscription // awaits ack on reply-to to be propagated to client
	awaitsClientAck map[uint16]*pkg.Publish       // awaits ack from client to be propagated to nats
	awaitsAckLock   sync.RWMutex
}

func (s *session) MarshalJSON() ([]byte, error) {
	return jsonstream.Marshal(s)
}

func (s *session) MarshalToJSON(w io.Writer) {
	pio.WriteString(`{"id":`, w)
	jsonstream.WriteString(s.id, w)
	pio.WriteString(`,"cid":`, w)
	jsonstream.WriteString(s.clientID, w)
	s.awaitsAckLock.RLock()
	if len(s.awaitsAck) > 0 {
		pio.WriteString(`,"awAck":`, w)
		sep := byte('{')
		for k, v := range s.awaitsAck {
			pio.WriteByte(sep, w)
			sep = byte(',')
			pio.WriteByte('"', w)
			pio.WriteInt(int64(k), w)
			pio.WriteString(`":`, w)
			jsonstream.WriteString(v.Subject, w)
		}
		pio.WriteByte('}', w)
	}
	if len(s.awaitsClientAck) > 0 {
		pio.WriteString(`,"awClientAck":`, w)
		sep := byte('{')
		for k, v := range s.awaitsClientAck {
			pio.WriteByte(sep, w)
			sep = byte(',')
			pio.WriteByte('"', w)
			pio.WriteInt(int64(k), w)
			pio.WriteString(`":`, w)
			v.MarshalToJSON(w)
		}
		pio.WriteByte('}', w)
	}
	pio.WriteByte('}', w)
	s.awaitsAckLock.RUnlock()
}

func (s *session) UnmarshalFromJSON(js *json.Decoder, t json.Token) {
	jsonstream.AssertDelimToken(t, '{')
	for {
		k, ok := jsonstream.AssertStringOrEnd(js, '}')
		if !ok {
			break
		}
		switch k {
		case "id":
			s.id = jsonstream.AssertString(js)
		case "cid":
			s.clientID = jsonstream.AssertString(js)
		case "awAck":
			jsonstream.AssertDelim(js, '{')
			for {
				k, ok = jsonstream.AssertStringOrEnd(js, '}')
				if !ok {
					break
				}
				if s.prelAwaitsAck == nil {
					s.prelAwaitsAck = make(map[uint16]string)
				}
				i, err := strconv.Atoi(k)
				if err != nil {
					panic(pio.Error{Cause: err})
				}
				s.prelAwaitsAck[uint16(i)] = jsonstream.AssertString(js)
			}
		case "awClientAck":
			jsonstream.AssertDelim(js, '{')
			for {
				k, ok = jsonstream.AssertStringOrEnd(js, '}')
				if !ok {
					break
				}
				if s.awaitsClientAck == nil {
					s.awaitsClientAck = make(map[uint16]*pkg.Publish)
				}
				i, err := strconv.Atoi(k)
				if err != nil {
					panic(pio.Error{Cause: err})
				}
				pp := &pkg.Publish{}
				jsonstream.AssertConsumer(js, pp)
				s.awaitsClientAck[uint16(i)] = pp
			}
		}
	}
}

func (s *session) RestoreAckSubscriptions(c *client) {
	if s.prelAwaitsAck != nil {
		for k, v := range s.prelAwaitsAck {
			sb, err := c.natsSubscribeAck(v)
			if err != nil {
				c.Error(err)
			} else {
				s.AckRequested(k, sb)
			}
		}
		s.prelAwaitsAck = nil
	}
}

func (s *session) AckReceived(packetID uint16) bool {
	awaits := false
	s.awaitsAckLock.Lock()
	if s.awaitsAck != nil {
		var sb *nats.Subscription
		if sb, awaits = s.awaitsAck[packetID]; awaits {
			_ = sb.Unsubscribe()
			delete(s.awaitsAck, packetID)
		}
	}
	s.awaitsAckLock.Unlock()
	return awaits
}

func (s *session) AckRequested(packetID uint16, sb *nats.Subscription) {
	s.awaitsAckLock.Lock()
	if s.awaitsAck == nil {
		s.awaitsAck = make(map[uint16]*nats.Subscription)
	}
	s.awaitsAck[packetID] = sb
	s.awaitsAckLock.Unlock()
}

func (s *session) AwaitsAck(packetID uint16) bool {
	awaits := false
	s.awaitsAckLock.RLock()
	if s.awaitsAck != nil {
		_, awaits = s.awaitsAck[packetID]
	}
	s.awaitsAckLock.RUnlock()
	return awaits
}

func (s *session) ClientAckReceived(packetID uint16, c *nats.Conn) bool {
	var pp *pkg.Publish
	s.awaitsAckLock.Lock()
	if s.awaitsClientAck != nil {
		var found bool
		if pp, found = s.awaitsClientAck[packetID]; found {
			delete(s.awaitsClientAck, packetID)
		}
	}
	s.awaitsAckLock.Unlock()
	if pp != nil {
		_ = c.Publish(pp.NatsReplyTo(), []byte{0})
		return true
	}
	return false
}

func (s *session) ClientAckRequested(pp *pkg.Publish) {
	s.awaitsAckLock.Lock()
	if s.awaitsClientAck == nil {
		s.awaitsClientAck = make(map[uint16]*pkg.Publish)
	}
	s.awaitsClientAck[pp.ID()] = pp
	s.awaitsAckLock.Unlock()
}

func (s *session) ResendClientUnack(c *client) {
	s.awaitsAckLock.RLock()
	as := make([]*pkg.Publish, 0, len(s.awaitsClientAck))
	for _, a := range s.awaitsClientAck {
		as = append(as, a)
	}
	s.awaitsAckLock.RUnlock()
	for i := range as {
		a := as[i]
		c.PublishResponse(a.QoSLevel(), a)
	}
}

func (s *session) ID() string {
	return s.id
}

func (s *session) ClientID() string {
	return s.clientID
}

func (s *session) Destroy() {
	// Unsubscribe all pending subscriptions
	s.awaitsAckLock.Lock()
	if s.awaitsAck != nil {
		for _, sb := range s.awaitsAck {
			_ = sb.Unsubscribe()
		}
		s.awaitsAck = nil
	}
	s.awaitsAckLock.Unlock()
}

type SessionManager interface {
	// Create creates a new session for the given clientID. Any previous session registered for
	// the given id is discarded
	Create(clientID string) Session

	// Get returns an existing session for the given clientID or nil if no such session exists
	Get(clientID string) Session

	// Remove removes any session for the given clientID
	Remove(clientID string)
}

type sm struct {
	lock sync.RWMutex
	seed uint32
	m    map[string]Session
}

func (m *sm) Get(clientID string) Session {
	var s Session
	m.lock.RLock()
	s = m.m[clientID]
	m.lock.RUnlock()
	return s
}

func (m *sm) Create(clientID string) Session {
	m.lock.Lock()
	m.seed++
	s := &session{id: `s` + strconv.Itoa(int(m.seed)), clientID: clientID}
	m.m[clientID] = s
	m.lock.Unlock()
	return s
}

func (m *sm) MarshalToJSON(w io.Writer) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	pio.WriteString(`{"seed":`, w)
	pio.WriteInt(int64(m.seed), w)
	if len(m.m) > 0 {
		pio.WriteString(`,"sessions":`, w)
		sep := byte('{')
		for k, v := range m.m {
			pio.WriteByte(sep, w)
			sep = ','
			jsonstream.WriteString(k, w)
			pio.WriteByte(':', w)
			v.MarshalToJSON(w)
		}
		pio.WriteByte('}', w)
	}
	pio.WriteByte('}', w)
}

func (m *sm) UnmarshalFromJSON(js *json.Decoder, t json.Token) {
	jsonstream.AssertDelimToken(t, '{')
	for {
		k, ok := jsonstream.AssertStringOrEnd(js, '}')
		if !ok {
			break
		}
		switch k {
		case "sessions":
			jsonstream.AssertDelim(js, '{')
			for {
				k, ok = jsonstream.AssertStringOrEnd(js, '}')
				if !ok {
					break
				}
				s := &session{}
				jsonstream.AssertConsumer(js, s)
				m.m[k] = s
			}
		case "seed":
			m.seed = uint32(jsonstream.AssertInt(js))
		}
	}
}

func (m *sm) Remove(clientID string) {
	var s Session
	m.lock.Lock()
	s = m.m[clientID]
	delete(m.m, clientID)
	m.lock.Unlock()
	if s != nil {
		s.Destroy()
	}
}
