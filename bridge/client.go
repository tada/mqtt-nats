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
	// StateInfant is set when the client is created and awaits a Connect package
	StateInfant = byte(iota)

	// StateConnected is set once a succesfull connection has been established
	StateConnected

	// StateDisconnected is set when a disconnect package arrives or when a non recoverable error occurs.
	StateDisconnected
)

// A Client represents a connection from a client.
type Client interface {
	// Serve starts the read and write loops and then waits for them to finish which
	// normally happens after the receipt of a disconnect package
	Serve()

	// State returns the current client state
	State() byte

	// PublishResponse publishes a package to the client in response to a subscription
	PublishResponse(qos byte, pp *pkg.Publish)

	// SetDisconnected will end the read and write loop and eventually cause Serve() to end.
	SetDisconnected(error)
}

type client struct {
	server         Server
	log            logger.Logger
	mqttConn       net.Conn
	natsOpts       *nats.Options
	natsConn       *nats.Conn
	session        Session
	connectPackage *pkg.Connect
	err            error
	natsSubs       map[string]*nats.Subscription
	writeQueue     chan pkg.Package
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
		writeQueue: make(chan pkg.Package, writeQueueSize)}
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

	cp := c.connectPackage
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
	if cp := c.connectPackage; cp != nil {
		return "Client " + cp.ClientID()
	}
	return "Client (not connected)"
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
		if cp := c.connectPackage; cp != nil && cp.HasWill() {
			w := cp.Will()
			err := c.publish(w.Topic, w.QoS, w.Retain, w.Message)
			if err != nil {
				c.Error(err)
			} else {
				c.Debug("will published to", w.Topic)
			}
		}
		// This package will not be sent but it will terminate the write loop once everything else
		// has been flushed
		c.writeQueue <- pkg.DisconnectSingleton

		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		c.err = err

		if err != io.ErrUnexpectedEOF {
			// release reader block
			_ = c.mqttConn.SetReadDeadline(time.Now().Add(time.Millisecond))
		}
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

func (c *client) Info(args ...interface{}) {
	if c.log.InfoEnabled() {
		c.log.Info(c.addFirst(args)...)
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

readNextPackage:
	for st := c.State(); st != StateDisconnected && err == nil; st = c.State() {
		var (
			b  byte
			rl int
		)

		if maxWait > 0 {
			_ = c.mqttConn.SetReadDeadline(time.Now().Add(maxWait))
		}

		// Read package type and flags
		if b, err = r.ReadByte(); err != nil {
			break
		}

		pkgType := b & pkg.TpMask
		switch st {
		case StateConnected:
			if pkgType == pkg.TpConnect {
				err = errors.New("second connect package")
				break readNextPackage
			}
		case StateInfant:
			if pkgType != pkg.TpConnect {
				err = errors.New("not connected")
				break readNextPackage
			}
		}

		// Read package length
		if rl, err = r.ReadVarInt(); err != nil {
			break
		}

		switch pkgType {
		case pkg.TpDisconnect:
			// Normal disconnect
			// Discard will
			c.Debug("received", pkg.DisconnectSingleton)
			c.connectPackage.DeleteWill()
			break readNextPackage
		case pkg.TpPing:
			pr := pkg.PingRequestSingleton
			c.Debug("received", pr)
			c.queueForWrite(pkg.PingResponseSingleton)
		case pkg.TpConnect:
			var cp *pkg.Connect
			if cp, err = pkg.ParseConnect(r, b, rl); err == nil {
				c.Debug("received", cp)
				maxWait, err = c.handleConnect(cp)
				if err == nil {
					c.server.ManageClient(c)
				}
			}
			if retCode, ok := err.(pkg.ReturnCode); ok {
				c.Debug("received", cp, "return code", retCode)
				c.setState(StateConnected)
				c.queueForWrite(pkg.NewAckConnect(c.sessionPresent, retCode))
			}
		case pkg.TpPublish:
			var pp *pkg.Publish
			if pp, err = pkg.ParsePublish(r, b, rl); err == nil {
				c.Debug("received", pp)
				err = c.natsPublish(c.server.HandleRetain(pp))
			}
		case pkg.TpPubAck:
			var pa pkg.PubAck
			if pa, err = pkg.ParsePubAck(r, b, rl); err == nil {
				c.Debug("received", pa)
				c.session.ClientAckReceived(pa.ID(), c.natsConn)
				c.server.ReleasePackageID(pa.ID())
			}
		case pkg.TpPubRec:
			var pa pkg.PubRec
			if pa, err = pkg.ParsePubRec(r, b, rl); err == nil {
				c.Debug("received", pa)
				// TODO: handle PubRec
			}
		case pkg.TpPubRel:
			var pa pkg.PubRel
			if pa, err = pkg.ParsePubRel(r, b, rl); err == nil {
				c.Debug("received", pa)
				// TODO: handle PubRel
			}
		case pkg.TpPubComp:
			var pa pkg.PubComp
			if pa, err = pkg.ParsePubComp(r, b, rl); err == nil {
				c.Debug("received", pa)
				// TODO: handle PubComp
			}
		case pkg.TpSubscribe:
			var sp *pkg.Subscribe
			if sp, err = pkg.ParseSubscribe(r, b, rl); err == nil {
				c.Debug("received", sp)
				if err = c.natsSubscribe(sp); err == nil {
					c.server.PublishMatching(sp, c)
				}
			}
		case pkg.TpUnsubscribe:
			var up *pkg.Unsubscribe
			if up, err = pkg.ParseUnsubscribe(r, b, rl); err == nil {
				c.Debug("received", up)
				c.natsUnsubscribe(up)
			}
		default:
			c.Debug("received unknown package type", (b&pkg.TpMask)>>4)
		}
	}
	c.SetDisconnected(err)
}

func (c *client) handleConnect(cp *pkg.Connect) (time.Duration, error) {
	var err error
	c.connectPackage = cp
	c.natsOpts, err = c.natsOptions()
	if err == nil {
		c.natsConn, err = c.natsOpts.Connect()
	}
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
		c.Debug("connected with clean session")
	} else {
		if s := m.Get(cid); s != nil {
			c.session = s
			c.sessionPresent = true
			c.Debug("connected using preexisting session")
			s.RestoreAckSubscriptions(c)
			s.ResendClientUnack(c)
		} else {
			c.Debug("connected using new (unclean) session")
			c.session = m.Create(cid)
		}
	}
	var maxWait time.Duration
	if cp.KeepAlive() > 0 {
		// Max wait between control packages is 1.5 times the keep alive value
		maxWait = (time.Duration(cp.KeepAlive()) * time.Second * 3) / 2
	}
	c.setState(StateConnected)
	c.queueForWrite(pkg.NewAckConnect(c.sessionPresent, 0))
	return maxWait, nil
}

func (c *client) queueForWrite(p pkg.Package) {
	if c.State() == StateConnected {
		c.writeQueue <- p
	}
}

func (c *client) writeLoop() {
	defer c.workers.Done()

	bulk := make([]pkg.Package, writeQueueSize)

	// writer's buffer is reused for each bulk operation
	w := mqtt.NewWriter()

	// Each iteration of this loop with pick max writeQueueSize packages from the writeQueue
	// and then write those packages on mqtt.Writer (a bytes.Buffer extension). The resulting bytes
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

func (c *client) publish(topic string, qos byte, retain bool, payload []byte) error {
	id := uint16(0)
	if qos > 0 {
		id = c.server.NextFreePackageID()
	}
	pp := pkg.NewPublish2(id, topic, payload, qos, false, retain)
	err := c.natsPublish(pp)
	if err == nil {
		if retain {
			c.server.HandleRetain(pp)
		} else if qos > 0 {
			pp.SetDup()
			c.server.TrackAckReceived(pp, c.connectPackage)
		}
	}
	return err
}
