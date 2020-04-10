package test

import (
	"os"
	"strconv"
	"testing"

	"github.com/tada/mqtt-nats/bridge"
	"github.com/tada/mqtt-nats/logger"
	"github.com/tada/mqtt-nats/test/full"
)

var mqttServer bridge.Bridge

const (
	storageFile          = "mqtt-nats.json"
	mqttPort             = 11883
	natsPort             = 14222
	retainedRequestTopic = "mqtt.retained.request"
)

func TestMain(m *testing.M) {
	_ = os.Remove(storageFile)
	natsServer := full.NATSServerOnPort(natsPort)

	// NOTE: Setting level to logger.Debug here is very helpful when authoring and debugging tests but
	//  it also makes the tests very verbose.
	lg := logger.New(logger.Debug, os.Stdout, os.Stderr)

	opts := bridge.Options{
		Port:                 mqttPort,
		NATSUrls:             ":" + strconv.Itoa(natsPort),
		RepeatRate:           50,
		RetainedRequestTopic: retainedRequestTopic,
		StoragePath:          storageFile}
	var err error
	mqttServer, err = full.RunBridge(lg, &opts)

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
