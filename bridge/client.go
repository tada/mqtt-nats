package bridge

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tada/mqtt-nats/logger"
	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/mqtt/pkg"
)

const (
	// StateInfant is set when the client is created and awaits a Connect packet
	StateInfant = byte(iota)

	// StateConnected is set once a successful connection has been established
	StateConnected

	// StateDisconnected is set when a disconnect packet arrives or when a non recoverable error occurs.
	StateDisconnected
)

// A Client represents a connection from a client.
type Client interface {
	// Serve starts the read and write loops and then waits for them to finish which
	// normally happens after the receipt of a disconnect packet
	Serve()

	// State returns the current client state
	State() byte

	// PublishResponse publishes a packet to the client in response to a subscription
	PublishResponse(qos byte, pp *pkg.Publish)

	// SetDisconnected will end the read and write loop and eventually cause Serve() to end.
	SetDisconnected(error)
}

type client struct {
	server         Server
	log            logger.Logger
	mqttConn       net.Conn
	natsConn       *nats.Conn
	session        Session
	connectPacket  *pkg.Connect
	err            error
	natsSubs       map[string]*nats.Subscription
	writeQueue     chan pkg.Packet
	stLock         sync.RWMutex
	subLock        sync.Mutex
	workers        sync.WaitGroup
	sessionPresent bool
	st             byte
}

// TODO: This should probably be configurable.
const writeQueueSize = 1024

// NewClient returns a new Client instance with StateInfant state.
func NewClient(s Server, log logger.Logger, conn net.Conn) Client {
	return &client{
		server:     s,
		log:        log,
		mqttConn:   conn,
		natsSubs:   make(map[string]*nats.Subscription),
		st:         StateInfant,
		writeQueue: make(chan pkg.Packet, writeQueueSize)}
}

func (c *client) Serve() {
	defer func() {
		_ = c.mqttConn.Close()

		if c.natsConn != nil {
			c.natsConn.Close()
		}
	}()

	c.workers.Add(2)
	go c.readLoop()
	go c.writeLoop()
	c.workers.Wait()

	cp := c.connectPacket
	if c.err != nil {
		c.Error(c.err)
	}
	if cp == nil {
		// No connection was established
		c.Debug("client connection could not be established")
	} else {
		c.Debug("disconnected")
		if cp.CleanSession() {
			c.server.SessionManager().Remove(cp.ClientID())
			c.Debug("session removed")
		}
	}
}

// String returns a text suitable for logging of client messages.
func (c *client) String() string {
	switch c.State() {
	case StateInfant:
		return "Client (not yet connected)"
	case StateConnected:
		return "Client " + c.connectPacket.ClientID()
	default:
		return "Client " + c.connectPacket.ClientID() + " (disconnected)"
	}
}

func (c *client) State() byte {
	var s byte
	c.stLock.RLock()
	s = c.st
	c.stLock.RUnlock()
	return s
}

func (c *client) setState(newState byte) {
	c.stLock.Lock()
	c.st = newState
	c.stLock.Unlock()
}

func (c *client) SetDisconnected(err error) {
	doit := false
	c.stLock.Lock()
	if c.st != StateDisconnected {
		doit = true
		c.st = StateDisconnected
	}
	c.stLock.Unlock()

	if doit {
		if cp := c.connectPacket; cp != nil && cp.HasWill() {
			err := c.server.PublishWill(cp.Will(), cp.Credentials())
			if err != nil {
				c.Error(err)
			} else {
				c.Debug("will published to", cp.Will().Topic)
			}
		}
		// This packet will not be sent but it will terminate the write loop once everything else
		// has been flushed
		c.writeQueue <- pkg.DisconnectSingleton

		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		c.err = err

		// release reader block
		_ = c.mqttConn.SetReadDeadline(time.Now().Add(time.Millisecond))
	}
}

func (c *client) Debug(args ...interface{}) {
	if c.log.DebugEnabled() {
		c.log.Debug(c.addFirst(args)...)
	}
}

func (c *client) Error(args ...interface{}) {
	if c.log.ErrorEnabled() {
		c.log.Error(c.addFirst(args)...)
	}
}

// addFirst prepends the client to the args slice and returns the new slice
func (c *client) addFirst(args []interface{}) []interface{} {
	na := make([]interface{}, len(args)+1)
	na[0] = c
	copy(na[1:], args)
	return na
}

func (c *client) readLoop() {
	defer c.workers.Done()

	r := mqtt.NewReader(c.mqttConn)

	var err error
	var maxWait time.Duration

readNextPacket:
	for st := c.State(); st != StateDisconnected && err == nil; st = c.State() {
		var (
			b  byte
			rl int
		)

		if maxWait > 0 {
			_ = c.mqttConn.SetReadDeadline(time.Now().Add(maxWait))
		}

		// Read packet type and flags
		if b, err = r.ReadByte(); err != nil {
			break
		}

		pkgType := b & pkg.TpMask
		switch st {
		case StateConnected:
			if pkgType == pkg.TpConnect {
				err = errors.New("second connect packet")
				break readNextPacket
			}
		case StateInfant:
			if pkgType != pkg.TpConnect {
				err = errors.New("not connected")
				break readNextPacket
			}
		}

		// Read packet length
		if rl, err = r.ReadVarInt(); err != nil {
			break
		}

		var p pkg.Packet
		switch pkgType {
		case pkg.TpDisconnect:
			// Normal disconnect
			// Discard will
			c.Debug("received", pkg.DisconnectSingleton)
			c.connectPacket.DeleteWill()
			break readNextPacket
		case pkg.TpPing:
			pr := pkg.PingRequestSingleton
			c.Debug("received", pr)
			c.queueForWrite(pkg.PingResponseSingleton)
		case pkg.TpConnect:
			if p, err = pkg.ParseConnect(r, b, rl); err == nil {
				c.Debug("received", p)
				maxWait, err = c.handleConnect(p.(*pkg.Connect))
				if err == nil {
					c.server.ManageClient(c)
				}
			}
			if retCode, ok := err.(pkg.ReturnCode); ok {
				c.Debug("received", p, "return code", retCode)
				c.setState(StateConnected)
				c.queueForWrite(pkg.NewConnAck(c.sessionPresent, retCode))
			}
		case pkg.TpPublish:
			if p, err = pkg.ParsePublish(r, b, rl); err == nil {
				c.Debug("received", p)
				err = c.natsPublish(c.server.HandleRetain(p.(*pkg.Publish)))
			}
		case pkg.TpPubAck:
			if p, err = pkg.ParsePubAck(r, b, rl); err == nil {
				c.Debug("received", p)
				id := p.(pkg.PubAck).ID()
				c.session.ClientAckReceived(id, c.natsConn)
				c.server.ReleasePacketID(id)
			}
		case pkg.TpPubRec:
			if p, err = pkg.ParsePubRec(r, b, rl); err == nil {
				c.Debug("received", p)
				// TODO: handle PubRec
			}
		case pkg.TpPubRel:
			if p, err = pkg.ParsePubRel(r, b, rl); err == nil {
				c.Debug("received", p)
				// TODO: handle PubRel
			}
		case pkg.TpPubComp:
			if p, err = pkg.ParsePubComp(r, b, rl); err == nil {
				c.Debug("received", p)
				// TODO: handle PubComp
			}
		case pkg.TpSubscribe:
			if p, err = pkg.ParseSubscribe(r, b, rl); err == nil {
				c.Debug("received", p)
				sp := p.(*pkg.Subscribe)
				if err = c.natsSubscribe(sp); err == nil {
					c.server.PublishMatching(sp, c)
				}
			}
		case pkg.TpUnsubscribe:
			if p, err = pkg.ParseUnsubscribe(r, b, rl); err == nil {
				c.Debug("received", p)
				c.natsUnsubscribe(p.(*pkg.Unsubscribe))
			}
		default:
			c.Debug("received unknown packet type", (b&pkg.TpMask)>>4)
		}
	}
	c.SetDisconnected(err)
}

func (c *client) handleConnect(cp *pkg.Connect) (time.Duration, error) {
	var err error
	c.connectPacket = cp
	c.natsConn, err = c.server.NatsConn(cp.Credentials())
	if err != nil {
		// TODO: Different error codes depending on error from NATS
		c.Error("NATS connect failed", err)
		return 0, pkg.RtServerUnavailable
	}

	cid := cp.ClientID()
	m := c.server.SessionManager()
	c.sessionPresent = false

	if cp.CleanSession() {
		c.session = m.Create(cid)
	} else {
		if s := m.Get(cid); s != nil {
			c.session = s
			c.sessionPresent = true
		} else {
			c.session = m.Create(cid)
		}
	}

	var maxWait time.Duration
	if cp.KeepAlive() > 0 {
		// Max wait between control packets is 1.5 times the keep alive value
		maxWait = (cp.KeepAlive() * 3) / 2
	}
	c.setState(StateConnected)
	c.queueForWrite(pkg.NewConnAck(c.sessionPresent, 0))

	if cp.CleanSession() {
		c.Debug("connected with clean session")
	} else {
		if c.sessionPresent {
			c.Debug("connected using preexisting session")
			c.session.RestoreAckSubscriptions(c)
			c.session.ResendClientUnack(c)
		} else {
			c.Debug("connected using new (unclean) session")
		}
	}
	return maxWait, nil
}

func (c *client) queueForWrite(p pkg.Packet) {
	if c.State() == StateConnected {
		c.writeQueue <- p
	}
}

func (c *client) writeLoop() {
	defer c.workers.Done()

	bulk := make([]pkg.Packet, writeQueueSize)

	// writer's buffer is reused for each bulk operation
	w := mqtt.NewWriter()

	// Each iteration of this loop with pick max writeQueueSize packets from the writeQueue
	// and then write those packets on mqtt.Writer (a bytes.Buffer extension). The resulting bytes
	// are then written to the connection using one single write on the connection.
	for connected := true; connected; {
		bulk[0] = <-c.writeQueue
		i := 1
	inner:
		for ; i < writeQueueSize; i++ {
			select {
			case p := <-c.writeQueue:
				bulk[i] = p
			default:
				break inner
			}
		}
		w.Reset()

		for n := 0; n < i; n++ {
			p := bulk[n]
			if p == pkg.DisconnectSingleton {
				connected = false
				break
			}
			c.Debug("sending", p)
			p.Write(w)
		}
		bs := w.Bytes()
		if len(bs) > 0 {
			if _, err := c.mqttConn.Write(bs); err != nil {
				if connected {
					c.SetDisconnected(err)
				} else {
					// Drain failed. Log the error
					c.Error(err)
				}
				break
			}
		}
	}
}

func (c *client) PublishResponse(qos byte, pp *pkg.Publish) {
	if qos > 0 {
		c.session.ClientAckRequested(pp)
	}
	c.queueForWrite(pp)
}
