package bridge

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nuid"
	"github.com/tada/mqtt-nats/jsonstream"
	"github.com/tada/mqtt-nats/logger"
	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/mqtt/pkg"
	"github.com/tada/mqtt-nats/pio"
)

type Server interface {
	pkg.IDManager
	SessionManager() SessionManager
	TrackAckReceived(pp *pkg.Publish, connectPackage *pkg.Connect)
	ManageClient(c Client)
	NatsOptions(user *string, password []byte) (*nats.Options, error)
	HandleRetain(pp *pkg.Publish) *pkg.Publish
	PublishMatching(sp *pkg.Subscribe, c Client)
	MessagesMatchingRetainRequest(m *nats.Msg) ([]*pkg.Publish, []byte)
}

type Bridge interface {
	Server
	Done() <-chan bool
	Restart(ready chan<- bool) error
	Serve(ready chan<- bool) error
	ServeClient(conn net.Conn)
	Shutdown() error
}

type server struct {
	logger.Logger
	pkg.IDManager
	opts             *Options
	session          Session
	retainedPackages *retained
	sm               SessionManager
	natsConn         *nats.Conn // servers NATS connection
	clients          []Client
	clientWG         sync.WaitGroup
	clientLock       sync.RWMutex
	trackAckLock     sync.RWMutex
	pubAcks          map[uint16]*natsPub // will be republished until ack arrives from nats
	pubAckTimeout    time.Duration
	pubAckTimer      *time.Timer
	done             chan bool
	signals          chan os.Signal
}

func New(opts *Options, logger logger.Logger) (Bridge, error) {
	s := &server{
		Logger:    logger,
		IDManager: pkg.NewIDManager(),
		opts:      opts,
		retainedPackages: &retained{
			msgs: make(map[string]*pkg.Publish),
		},
		pubAckTimeout: time.Duration(opts.RepeatRate) * time.Millisecond,
		sm:            &sm{m: make(map[string]Session, 37)},
		signals:       make(chan os.Signal, 1),
	}

	s.session = s.sm.Create(`mqtt-nats-` + nuid.Next())
	var err error
	if opts.StoragePath != "" {
		err = s.load(opts.StoragePath)
	}
	return s, err
}

func (s *server) Restart(ready chan<- bool) error {
	err := s.Shutdown()
	if err == nil && s.opts.StoragePath != "" {
		err = s.load(s.opts.StoragePath)
	}
	if err == nil {
		err = s.Serve(ready)
	}
	return err
}

func (s *server) Shutdown() error {
	s.signals <- syscall.SIGINT
	select {
	case <-s.Done():
	case <-time.After(5 * time.Millisecond):
		return errors.New("timeout during bridge shutdown")
	}
	return nil
}

// Done will be closed when the shutdown is complete
func (s *server) Done() <-chan bool {
	return s.done
}

func (s *server) Serve(ready chan<- bool) error {
	s.done = make(chan bool, 1)

	listener, err := getTCPListener(s.opts)
	if err != nil {
		if ready != nil {
			ready <- false
		}
		return err
	}

	if s.opts.RetainedRequestTopic != "" {
		err = s.startRetainedRequestHandler()
		if err != nil {
			return err
		}
	}

	signal.Notify(s.signals, syscall.SIGINT, syscall.SIGTERM)
	shuttingDown := false

	go func() {
		sig := <-s.signals
		s.Debug("trapped signal", sig)
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			s.Debug("mqtt-nats is shutting down")
			shuttingDown = true
			_ = listener.Close()
		}
		// TODO: Add signal to reload config.
	}()

	if ready != nil {
		ready <- true
	}
	for {
		mqttConn, err := listener.Accept()
		if err != nil {
			if shuttingDown {
				// error due to close of listener
				break
			}
			s.Error(err)
		} else {
			go s.ServeClient(mqttConn)
		}
	}
	return s.drainAndShutdown()
}

func (s *server) drainAndShutdown() error {
	s.Debug("waiting for clients to drain")
	s.clientLock.Lock()
	clients := s.clients
	s.clients = nil
	s.clientLock.Unlock()

	for i := range clients {
		clients[i].SetDisconnected(nil)
	}
	s.clientWG.Wait()
	s.Debug("client drain complete")

	s.trackAckLock.Lock()
	if s.pubAckTimer != nil {
		s.pubAckTimer.Stop()
		s.pubAckTimer = nil
	}
	s.trackAckLock.Unlock()

	if s.natsConn != nil {
		s.natsConn.Close()
		s.natsConn = nil
	}

	var err error
	if s.opts.StoragePath != "" {
		err = s.persist(s.opts.StoragePath)
	}
	close(s.done)
	return err
}

func (s *server) startRetainedRequestHandler() error {
	conn, err := s.getNatsConn()
	if err == nil {
		_, err = conn.Subscribe(s.opts.RetainedRequestTopic, s.handleRetainedRequest)
	}
	return err
}

func (s *server) handleRetainedRequest(m *nats.Msg) {
	pps, qs := s.MessagesMatchingRetainRequest(m)
	if len(pps) == 0 {
		return
	}
	err := pio.Catch(func() error {
		qos := byte(0)
		buf := &bytes.Buffer{}
		pio.WriteByte('[', buf)
		for i := range pps {
			pp := pps[i]
			if qs[0] > qos {
				qos = qs[0]
			}
			if i > 0 {
				pio.WriteByte(',', buf)
			}
			pio.WriteString(`{"subject":`, buf)
			jsonstream.WriteString(mqtt.ToNATS(pp.TopicName()), buf)
			pls := string(pp.Payload())
			if utf8.ValidString(pls) {
				pio.WriteString(`,"payload":`, buf)
				jsonstream.WriteString(pls, buf)
			} else {
				pio.WriteString(`,"payload_enc":`, buf)
				jsonstream.WriteString(base64.StdEncoding.EncodeToString(pp.Payload()), buf)
			}
			pio.WriteByte('}', buf)
		}
		pio.WriteByte(']', buf)
		return m.Respond(buf.Bytes())
	})
	if err != nil {
		s.Error("NATS publish of retained messages failed", err)
	}
}

func (s *server) MessagesMatchingRetainRequest(m *nats.Msg) ([]*pkg.Publish, []byte) {
	return s.retainedPackages.messagesMatchingRetainRequest(m)
}

func (s *server) NatsOptions(user *string, password []byte) (*nats.Options, error) {
	opts := nats.GetDefaultOptions()
	opts.Servers = strings.Split(s.opts.NATSUrls, ",")
	optFuncs := s.opts.NATSOpts
	for i := range optFuncs {
		if err := optFuncs[i](&opts); err != nil {
			return nil, err
		}
	}
	if user != nil {
		opts.User = *user
	}
	if password != nil {
		// TODO: Handle password correctly, definitions differ
		opts.Password = string(password)
	}
	return &opts, nil
}

// TrackAckReceived is used when a publish using QoS > 0 originates from this server and needs to be
// maintained until an ack is received. One example of when this happens is when a client will has QoS > 0
func (s *server) TrackAckReceived(pp *pkg.Publish, cp *pkg.Connect) {
	s.Debug("track", pp)
	s.trackAckLock.Lock()
	np := natsPub{pp: pp}
	if cp.HasUserName() {
		n := cp.Username()
		np.user = &n
	}
	if cp.HasPassword() {
		np.password = cp.Password()
	}
	if s.pubAcks == nil {
		s.pubAcks = make(map[uint16]*natsPub)
		s.pubAckTimer = time.AfterFunc(s.pubAckTimeout, s.ackCheckTick)
	}
	s.pubAcks[pp.ID()] = &np
	s.trackAckLock.Unlock()
}

func (s *server) ackCheckTick() {
	s.pubAckTimer.Reset(s.pubAckTimeout)
	for _, np := range s.awaitsAckSnapshot() {
		s.republish(np)
	}
}

func (s *server) awaitsAckSnapshot() []*natsPub {
	i := 0
	s.trackAckLock.RLock()
	nps := make([]*natsPub, len(s.pubAcks))
	for _, np := range s.pubAcks {
		nps[i] = np
		i++
	}
	s.trackAckLock.RUnlock()
	sort.Slice(nps, func(i, j int) bool { return nps[i].pp.ID() < nps[j].pp.ID() })
	return nps
}

func (s *server) getNatsConn() (*nats.Conn, error) {
	var err error
	if s.natsConn == nil {
		var natsOpts *nats.Options
		natsOpts, err = s.NatsOptions(nil, nil)
		if err == nil {
			s.natsConn, err = natsOpts.Connect()
		}
	}
	return s.natsConn, err
}

func (s *server) republish(np *natsPub) {
	conn, err := s.getNatsConn()
	if err != nil {
		s.Error(err)
		return
	}

	pp := np.pp
	replyTo := NewReplyTopic(s.session, pp).String()
	sub, err := conn.SubscribeSync(replyTo)
	if err != nil {
		s.Error(err)
		return
	}

	s.Debug("republish", pp)
	if err = conn.PublishRequest(mqtt.ToNATS(pp.TopicName()), replyTo, pp.Payload()); err != nil {
		s.Error(err)
		return
	}

	if _, err = sub.NextMsg(2 * time.Second); err == nil {
		s.Debug("ack", pp.ID())
		s.trackAckLock.Lock()
		if s.pubAcks != nil {
			delete(s.pubAcks, pp.ID())
			if len(s.pubAcks) == 0 {
				s.pubAckTimer.Stop()
				s.pubAckTimer = nil
				s.pubAcks = nil
			}
		}
		s.trackAckLock.Unlock()
	} else if err != nats.ErrTimeout {
		s.Error(err)
	}
}

func (s *server) MarshalToJSON(w io.Writer) {
	pio.WriteString(`{"ts":`, w)
	jsonstream.WriteString(time.Now().Format(time.RFC3339), w)
	pio.WriteString(`,"id":`, w)
	jsonstream.WriteString(s.session.ClientID(), w)
	pio.WriteString(`,"id_manager":`, w)
	s.IDManager.MarshalToJSON(w)
	pio.WriteString(`,"session_manager":`, w)
	s.sm.MarshalToJSON(w)
	if !s.retainedPackages.Empty() {
		pio.WriteString(`,"retained":`, w)
		s.retainedPackages.MarshalToJSON(w)
	}
	trk := s.awaitsAckSnapshot()
	if len(trk) > 0 {
		pio.WriteString(`,"pubacks":[`, w)
		for i := range trk {
			if i > 0 {
				pio.WriteByte(',', w)
			}
			trk[i].MarshalToJSON(w)
		}
		pio.WriteByte(']', w)
	}
	pio.WriteByte('}', w)
}

func (s *server) UnmarshalFromJSON(js *json.Decoder, t json.Token) {
	jsonstream.AssertDelimToken(t, '{')
	var (
		id        string
		ackTracks map[uint16]*natsPub
	)
	for {
		k, ok := jsonstream.AssertStringOrEnd(js, '}')
		if !ok {
			break
		}
		switch k {
		case "id":
			id = jsonstream.AssertString(js)
		case "id_manager":
			jsonstream.AssertConsumer(js, s.IDManager)
		case "session_manager":
			jsonstream.AssertConsumer(js, s.sm)
		case "retained":
			jsonstream.AssertConsumer(js, s.retainedPackages)
		case "pubacks":
			jsonstream.AssertDelim(js, '[')
			ackTracks = make(map[uint16]*natsPub)
			for {
				np := &natsPub{}
				if !jsonstream.AssertConsumerOrEnd(js, np, ']') {
					break
				}
				ackTracks[np.pp.ID()] = np
			}
		}
	}
	if id != "" {
		s.session = s.sm.Get(id)
	}
	if len(ackTracks) > 0 {
		s.pubAcks = ackTracks
		s.pubAckTimer = time.AfterFunc(s.pubAckTimeout, s.ackCheckTick)
	}
}

func (s *server) HandleRetain(pp *pkg.Publish) *pkg.Publish {
	if pp.Retain() {
		rt := s.retainedPackages
		if len(pp.Payload()) == 0 {
			if rt.drop(pp.TopicName()) {
				s.Debug("deleted retained message", pp)
			}
			pp.ResetRetain()
		} else if rt.add(pp) {
			s.Debug("added retained message", pp)
		}
	}
	return pp
}

func (s *server) PublishMatching(ps *pkg.Subscribe, c Client) {
	s.retainedPackages.publishMatching(ps, c)
}

func (s *server) ServeClient(conn net.Conn) {
	s.clientWG.Add(1)
	defer s.clientWG.Done()
	c := NewClient(s, s, conn)
	c.Serve()
	s.unmanageClient(c)
}

// ManageClient adds the client to the list of clients managed by the server
func (s *server) ManageClient(c Client) {
	s.clientLock.Lock()
	s.clients = append(s.clients, c)
	s.clientLock.Unlock()
}

// unmanageClient removes the client from the list of clients managed by the server
func (s *server) unmanageClient(c Client) {
	s.clientLock.Lock()
	cs := s.clients
	ln := len(cs) - 1
	for i := 0; i <= ln; i++ {
		if c == cs[i] {
			cs[i] = cs[ln]
			cs[ln] = nil
			s.clients = cs[:ln]
			break
		}
	}
	s.clientLock.Unlock()
}

func (s *server) load(path string) error {
	f, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return err
	}
	defer func() {
		err = f.Close()
	}()
	return pio.Catch(func() error {
		dc := json.NewDecoder(bufio.NewReader(f))
		dc.UseNumber()
		jsonstream.AssertConsumer(dc, s)
		s.Debug("State loaded from", path)
		return nil
	})
}

func (s *server) persist(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() {
		err = f.Close()
	}()
	return pio.Catch(func() error {
		w := bufio.NewWriter(f)
		s.MarshalToJSON(w)
		s.Debug("server State persisted to ", path)
		return w.Flush()
	})
}

func getTCPListener(opts *Options) (net.Listener, error) {
	ps := `:` + strconv.Itoa(opts.Port)
	if !opts.TLS {
		return net.Listen("tcp", ps)
	}

	// Load mandatory server key pair for bridge
	cer, err := tls.LoadX509KeyPair(opts.TLSCert, opts.TLSKey)
	if err != nil {
		return nil, err
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cer}}

	if opts.TLSVerify {
		cfg.ClientAuth = tls.RequestClientCert
	}

	if opts.TLSCaCert != `` {
		roots := x509.NewCertPool()
		var caPem []byte
		if caPem, err = ioutil.ReadFile(opts.TLSCaCert); err != nil {
			return nil, err
		}
		if ok := roots.AppendCertsFromPEM(caPem); !ok {
			return nil, fmt.Errorf("failed to parse root certificate in file: %s", opts.TLSCaCert)
		}
		cfg.ClientCAs = roots
	}

	return tls.Listen(`tcp`, ps, cfg)
}

func (s *server) SessionManager() SessionManager {
	return s.sm
}
