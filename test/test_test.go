package test

import (
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	testserver "github.com/nats-io/nats-server/v2/test"
	"github.com/tada/mqtt-nats/bridge"
	"github.com/tada/mqtt-nats/logger"
)

var mqttServer bridge.Bridge

const storageFile = "mqtt-nats.json"

func TestMain(m *testing.M) {
	_ = os.Remove(storageFile)
	natsServer := NATSServerOnPort(natsPort)

	// NOTE: Setting level to logger.Debug here is very helpful when authoring and debugging tests but
	//  it also makes the tests very verbose.
	lg := logger.New(logger.Silent, os.Stdout, os.Stderr)

	opts := bridge.Options{
		Port:                 mqttPort,
		NATSUrls:             ":" + strconv.Itoa(natsPort),
		RepeatRate:           50,
		RetainedRequestTopic: retainedRequestTopic,
		StoragePath:          storageFile}
	var err error
	mqttServer, err = RunBridgeOnPorts(lg, &opts)

	var code int
	if err == nil {
		code = m.Run()
	} else {
		lg.Error(err)
		code = 1
	}
	natsServer.Shutdown()
	if err = mqttServer.Shutdown(); err != nil {
		lg.Error(err)
		code = 1
	}
	os.Exit(code)
}

const mqttPort = 11883
const natsPort = 14222
const retainedRequestTopic = "mqtt.retained.request"

func RunBridgeOnPorts(lg logger.Logger, opts *bridge.Options) (bridge.Bridge, error) {
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
	return srv, nil
}

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
	return NATSServerWithOptions(opts)
}

// NATSServerWithOptions will run a server with the given options.
func NATSServerWithOptions(opts server.Options) *server.Server {
	return testserver.RunServer(&opts)
}

func assertMessageReceived(t *testing.T, c <-chan bool) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf(`expected packet did not arrive`)
	}
}

func assertTimeout(t *testing.T, c <-chan bool) {
	t.Helper()
	select {
	case <-c:
		t.Fatalf(`unexpected packet arrived`)
	case <-time.After(10 * time.Millisecond):
	}
}
