// +build citest

// Package full contains the test utilities that enables full roundtrip testing with both an mqtt-bridge and
// a NATS test server.
package full

import (
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	testserver "github.com/nats-io/nats-server/v2/test"
	"github.com/tada/mqtt-nats/bridge"
	"github.com/tada/mqtt-nats/logger"
)

// RunBridge starts an in-process mqtt-nats bridge configured with the given options.
func RunBridge(lg logger.Logger, opts *bridge.Options) (bridge.Bridge, error) {
	srv, err := bridge.New(opts, lg)
	if err != nil {
		return nil, err
	}

	serverReady := sync.WaitGroup{}
	serverReady.Add(1)
	go func() {
		err = srv.Serve(&serverReady)
		if err != nil {
			lg.Error(err)
		}
	}()
	serverReady.Wait()
	return srv, err
}

// RestartBridge restarts the given bridge
func RestartBridge(t *testing.T, b bridge.Bridge) {
	serverReady := sync.WaitGroup{}
	serverReady.Add(1)
	go func() {
		if err := b.Restart(&serverReady); err != nil {
			t.Error(err)
		}
	}()
	serverReady.Wait()
}

// NATSServerOnPort will run a server on the given port.
func NATSServerOnPort(port int) *server.Server {
	opts := testserver.DefaultTestOptions
	opts.Port = port
	return NATSServerWithOptions(&opts)
}

// NATSServerWithOptions will run a server with the given options.
func NATSServerWithOptions(opts *server.Options) *server.Server {
	return testserver.RunServer(opts)
}

// AssertMessageReceived waits for a boolean to be received on the given channel for one second and then
// bails out with a Fatal message "expected message did not arrive"
func AssertMessageReceived(t *testing.T, c <-chan bool) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(time.Second): // Wait time is somewhat arbitrary.
		t.Fatalf(`expected message did not arrive`)
	}
}

// AssertTimeout waits for 10 milliseconds to ensure that no boolean is received on the given channel and
// then bails out with a Fatal message "unexpected message arrived".
func AssertTimeout(t *testing.T, c <-chan bool) {
	t.Helper()
	select {
	case <-c:
		t.Fatalf(`unexpected message arrived`)
	case <-time.After(10 * time.Millisecond):
	}
}
