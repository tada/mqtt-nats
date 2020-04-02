package test

import (
	"os"
	"strconv"
	"testing"
	"time"

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

func assertMessageReceived(t *testing.T, c <-chan bool) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(10 * time.Millisecond):
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
