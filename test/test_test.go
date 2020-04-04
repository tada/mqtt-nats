package test

import (
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	testserver "github.com/nats-io/nats-server/v2/test"
	"github.com/tada/mqtt-nats/bridge"
	"github.com/tada/mqtt-nats/logger"
)

var mqttServer bridge.Bridge

func TestMain(m *testing.M) {
	natsServer := NATSServerOnPort(natsPort)

	// NOTE: Setting level to logger.Debug here is very helpful when authoring and debugging tests but
	//  it also makes the tests very verbose.
	lg := logger.New(logger.Debug, os.Stdout, os.Stderr)

	opts := bridge.Options{
		Port:                 mqttPort,
		NATSUrls:             ":" + strconv.Itoa(natsPort),
		RepeatRate:           50,
		RetainedRequestTopic: retainedRequestTopic,
		StoragePath:          "mqtt-nats.json"}
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

	serverReady := make(chan bool, 1)
	go func() {
		err = srv.Serve(serverReady)
		if err != nil {
			lg.Error(err)
		}
	}()

	if !<-serverReady {
		return nil, errors.New("mqtt-nats failed to start")
	}
	return srv, nil
}

func RestartBridge(t *testing.T, b bridge.Bridge) {
	ready := make(chan bool, 1)
	go func() {
		if err := mqttServer.Restart(ready); err != nil {
			t.Fatal(err)
		}
	}()

	if !<-ready {
		t.Fatal("mqtt-nats failed to start")
	}
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
	case <-time.After(time.Second):
		t.Fatalf(`expected package did not arrive`)
	}
}

func assertTimeout(t *testing.T, c <-chan bool) {
	t.Helper()
	select {
	case <-c:
		t.Fatalf(`unexpected package arrived`)
	case <-time.After(10 * time.Millisecond):
	}
}
