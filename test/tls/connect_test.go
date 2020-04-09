package tls

import (
	"testing"

	"github.com/tada/mqtt-nats/mqtt/pkg"
	"github.com/tada/mqtt-nats/test/full"
)

func TestConnect(t *testing.T) {
	conn := tlsDial(t, mqttPort)
	full.MqttSend(t, conn, pkg.NewConnect(full.NextClientID(), true, 1, nil, nil))
	full.MqttExpect(t, conn, pkg.NewConnAck(false, 0))
	full.MqttDisconnect(t, conn)
}
