// Package test contains the in-process functional tests for the mqtt-nats bridge
package test

import (
	"errors"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	testserver "github.com/nats-io/nats-server/v2/test"
	"github.com/tada/mqtt-nats/bridge"
	"github.com/tada/mqtt-nats/logger"
)

const mqttPort = 11883
const natsPort = 14222
const retainedRequestTopic = "mqtt.retained.request"

func RunBridgeOnPorts(lg logger.Logger, opts *bridge.Options) (bridge.Bridge, error) {
	server, err := bridge.New(opts, lg)
	if err != nil {
		return nil, err
	}

	serverReady := make(chan bool, 1)
	go func() {
		err = server.Serve(serverReady)
		if err != nil {
			lg.Error(err)
		}
	}()

	if !<-serverReady {
		return nil, errors.New("mqtt-nats failed to start")
	}
	return server, nil
}

func shutdown(server bridge.Bridge) {
	server.InitiateShutdown()
	select {
	case <-server.Done():
	case <-time.After(100 * time.Millisecond):
		server.(logger.Logger).Error("timeout during bridge shutdown")
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

// NATSServerWithConfig will run a server with the given configuration file.
func NATSServerWithConfig(configFile string) (*server.Server, *server.Options) {
	return testserver.RunServerWithConfig(configFile)
}
