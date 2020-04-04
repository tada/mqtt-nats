package test

import (
	"bytes"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/tada/mqtt-nats/mqtt/pkg"
)

func TestPublishSubscribe(t *testing.T) {
	topic := "testing/some/topic"
	pp := pkg.SimplePublish(topic, []byte("payload"))
	gotIt := make(chan bool, 1)
	c1 := mqttConnectClean(t, mqttPort)
	mid := nextPackageID()
	mqttSend(t, c1, pkg.NewSubscribe(mid, pkg.Topic{Name: topic}))
	mqttExpect(t, c1, pkg.NewSubAck(mid, 0))
	go func() {
		mqttExpect(t, c1, pp)
		gotIt <- true
		mqttDisconnect(t, c1)
	}()

	c2 := mqttConnectClean(t, mqttPort)
	mqttSend(t, c2, pp)
	mqttDisconnect(t, c2)
	assertMessageReceived(t, gotIt)
}

func TestPublishSubscribe_qos_1(t *testing.T) {
	topic := "testing/some/topic"
	mid := nextPackageID()
	pp := pkg.NewPublish2(mid, topic, []byte("payload"), 1, false, false)
	c1 := mqttConnectClean(t, mqttPort)
	gotIt := make(chan bool, 1)
	go func() {
		sid := nextPackageID()
		mqttSend(t, c1, pkg.NewSubscribe(sid, pkg.Topic{Name: topic, QoS: 1}))
		mqttExpect(t, c1, pkg.NewSubAck(sid, 1), pp)
		mqttSend(t, c1, pkg.PubAck(mid))
		mqttDisconnect(t, c1)
		gotIt <- true
	}()

	c2 := mqttConnectClean(t, mqttPort)
	mqttSend(t, c2, pp)
	mqttExpect(t, c2, pkg.PubAck(mid))
	mqttDisconnect(t, c2)
	assertMessageReceived(t, gotIt)
}

func TestPublishSubscribe_qos_1_restart(t *testing.T) {
	topic := "testing/some/topic"
	mid := nextPackageID()
	pp := pkg.NewPublish2(mid, topic, []byte("payload"), 1, false, false)

	c1ID := nextClientID()
	c1 := mqttConnect(t, mqttPort)
	gotIt := make(chan bool, 1)
	go func() {
		mqttSend(t, c1, pkg.NewConnect(c1ID, false, 1, nil, nil))
		mqttExpect(t, c1, pkg.NewAckConnect(false, 0))

		sid := nextPackageID()
		mqttSend(t, c1, pkg.NewSubscribe(sid, pkg.Topic{Name: topic, QoS: 1}))
		mqttExpect(t, c1, pkg.NewSubAck(sid, 1), pp)
		gotIt <- true
	}()

	c2ID := nextClientID()
	c2 := mqttConnect(t, mqttPort)
	mqttSend(t, c2, pkg.NewConnect(c2ID, false, 1, nil, nil))
	mqttExpect(t, c2, pkg.NewAckConnect(false, 0))
	mqttSend(t, c2, pp)

	assertMessageReceived(t, gotIt)

	RestartBridge(t, mqttServer)

	c1 = mqttConnect(t, mqttPort)
	mqttSend(t, c1, pkg.NewConnect(c1ID, false, 1, nil, nil))
	mqttExpect(t, c1, pkg.NewAckConnect(true, 0))

	c2 = mqttConnect(t, mqttPort)
	mqttSend(t, c2, pkg.NewConnect(c2ID, false, 1, nil, nil))
	mqttExpect(t, c2, pkg.NewAckConnect(true, 0))

	mqttSend(t, c1, pkg.PubAck(mid))
	mqttDisconnect(t, c1)

	mqttExpect(t, c2, pkg.PubAck(mid))
	mqttDisconnect(t, c2)
}

func TestMqttPublishNatsSubscribe(t *testing.T) {
	pl := []byte("payload")
	pp := pkg.SimplePublish("testing/some/topic", pl)
	gotIt := make(chan bool, 1)
	nc := natsConnect(t, natsPort)
	defer nc.Close()

	_, err := nc.Subscribe("testing.some.topic", func(m *nats.Msg) {
		if !bytes.Equal(pl, m.Data) {
			t.Error("nats subscription did not receive expected data")
		}
		gotIt <- true
	})
	if err != nil {
		t.Fatal(err)
	}

	c2 := mqttConnectClean(t, mqttPort)
	mqttSend(t, c2, pp)
	mqttDisconnect(t, c2)
	assertMessageReceived(t, gotIt)
}

func TestNatsPublishMqttSubscribe(t *testing.T) {
	topic := "testing/some/topic"
	pl := []byte("payload")
	pp := pkg.SimplePublish(topic, pl)

	c1 := mqttConnectClean(t, mqttPort)
	sid := nextPackageID()
	mqttSend(t, c1, pkg.NewSubscribe(sid, pkg.Topic{Name: topic}))
	mqttExpect(t, c1, pkg.NewSubAck(sid, 0))

	gotIt := make(chan bool, 1)
	go func() {
		mqttExpect(t, c1, pp)
		mqttDisconnect(t, c1)
		gotIt <- true
	}()

	nc := natsConnect(t, natsPort)
	defer nc.Close()
	err := nc.Publish("testing.some.topic", pl)
	if err != nil {
		t.Fatal(err)
	}
	assertMessageReceived(t, gotIt)
}

func TestUnubscribe(t *testing.T) {
	topic := "testing/some/topic"
	pp := pkg.SimplePublish(topic, []byte("payload"))
	gotIt := make(chan bool, 1)
	c1 := mqttConnectClean(t, mqttPort)
	mid := nextPackageID()
	mqttSend(t, c1, pkg.NewSubscribe(mid, pkg.Topic{Name: topic}))
	mqttExpect(t, c1, pkg.NewSubAck(mid, 0))
	go func() {
		mqttExpect(t, c1, pp)

		uid := nextPackageID()
		mqttSend(t, c1, pkg.NewUnsubscribe(uid, topic))
		mqttExpect(t, c1, pkg.UnsubAck(uid))
		gotIt <- true
		mqttExpectConnClosed(t, c1)
		gotIt <- true
	}()

	c2 := mqttConnectClean(t, mqttPort)
	mqttSend(t, c2, pp)

	// wait for subscriber to consume and unsubscribe
	assertMessageReceived(t, gotIt)

	// send again, this should not reach subscriber
	mqttSend(t, c2, pp)
	mqttDisconnect(t, c2)
	assertTimeout(t, gotIt)
	mqttDisconnect(t, c1)
}
