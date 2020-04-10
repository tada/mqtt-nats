package test

import (
	"bytes"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tada/mqtt-nats/mqtt/pkg"
	"github.com/tada/mqtt-nats/test/full"
)

func TestPublishSubscribe(t *testing.T) {
	topic := "testing/some/topic"
	pp := pkg.SimplePublish(topic, []byte("payload"))
	gotIt := make(chan bool, 1)
	c1 := full.MqttConnectClean(t, mqttPort)
	mid := nextPacketID()
	full.MqttSend(t, c1, pkg.NewSubscribe(mid, pkg.Topic{Name: topic}))
	full.MqttExpect(t, c1, pkg.NewSubAck(mid, 0))
	go func() {
		full.MqttExpect(t, c1, pp)
		gotIt <- true
		full.MqttDisconnect(t, c1)
	}()

	c2 := full.MqttConnectClean(t, mqttPort)
	full.MqttSend(t, c2, pp)
	full.MqttDisconnect(t, c2)
	full.AssertMessageReceived(t, gotIt)
}

func TestPublishSubscribe_qos_1(t *testing.T) {
	topic := "testing/some/topic"
	mid := nextPacketID()
	pp := pkg.NewPublish2(mid, topic, []byte("payload"), 1, false, false)
	c1 := full.MqttConnectClean(t, mqttPort)
	gotIt := make(chan bool, 1)
	go func() {
		sid := nextPacketID()
		full.MqttSend(t, c1, pkg.NewSubscribe(sid, pkg.Topic{Name: topic, QoS: 1}))
		full.MqttExpect(t, c1, pkg.NewSubAck(sid, 1), pp)
		full.MqttSend(t, c1, pkg.PubAck(mid))
		full.MqttDisconnect(t, c1)
		gotIt <- true
	}()

	c2 := full.MqttConnectClean(t, mqttPort)
	full.MqttSend(t, c2, pp)
	full.MqttExpect(t, c2, pkg.PubAck(mid))
	full.MqttDisconnect(t, c2)
	full.AssertMessageReceived(t, gotIt)
}

func TestPublishSubscribe_qos_2(t *testing.T) {
	topic := "testing/some/topic"
	mid := nextPacketID()
	pp := pkg.NewPublish2(mid, topic, []byte("payload"), 1, false, false)
	c1 := full.MqttConnectClean(t, mqttPort)
	gotIt := make(chan bool, 1)
	go func() {
		sid := nextPacketID()
		full.MqttSend(t, c1, pkg.NewSubscribe(sid, pkg.Topic{Name: topic, QoS: 2}))
		full.MqttExpect(t, c1, pkg.NewSubAck(sid, 1), pp)
		full.MqttSend(t, c1, pkg.PubAck(mid))
		full.MqttDisconnect(t, c1)
		gotIt <- true
	}()

	c2 := full.MqttConnectClean(t, mqttPort)
	full.MqttSend(t, c2, pp)
	full.MqttExpect(t, c2, pkg.PubAck(mid))
	full.MqttDisconnect(t, c2)
	full.AssertMessageReceived(t, gotIt)
}

func TestPublishSubscribe_qos_1_restart(t *testing.T) {
	topic := "testing/some/topic"
	mid := nextPacketID()
	pp := pkg.NewPublish2(mid, topic, []byte("payload"), 1, false, false)

	c1ID := full.NextClientID()
	c1 := full.MqttConnect(t, mqttPort)
	gotIt := make(chan bool, 1)
	go func() {
		full.MqttSend(t, c1, pkg.NewConnect(c1ID, false, 1, nil, nil))
		full.MqttExpect(t, c1, pkg.NewConnAck(false, 0))

		sid := nextPacketID()
		full.MqttSend(t, c1, pkg.NewSubscribe(sid, pkg.Topic{Name: topic, QoS: 1}))
		full.MqttExpect(t, c1, pkg.NewSubAck(sid, 1))
		gotIt <- true
		full.MqttExpect(t, c1, pp)
		gotIt <- true
	}()

	c2ID := full.NextClientID()
	c2 := full.MqttConnect(t, mqttPort)
	full.MqttSend(t, c2, pkg.NewConnect(c2ID, false, 1, nil, nil))
	full.MqttExpect(t, c2, pkg.NewConnAck(false, 0))
	full.AssertMessageReceived(t, gotIt)
	full.MqttSend(t, c2, pp)
	full.AssertMessageReceived(t, gotIt)

	full.RestartBridge(t, mqttServer)

	// client c1 reestablishes session and sends outstanding ack
	c1 = full.MqttConnect(t, mqttPort)
	full.MqttSend(t, c1, pkg.NewConnect(c1ID, false, 1, nil, nil))
	full.MqttExpect(t, c1, pkg.NewConnAck(true, 0))

	// client c2 reestablishes session and receives outstanding ack
	c2 = full.MqttConnect(t, mqttPort)
	full.MqttSend(t, c2, pkg.NewConnect(c2ID, false, 1, nil, nil))
	full.MqttExpect(t, c2, pkg.NewConnAck(true, 0))

	full.MqttSend(t, c1, pkg.PubAck(mid))
	full.MqttExpect(t, c2, pkg.PubAck(mid))

	full.MqttDisconnect(t, c1)
	full.MqttDisconnect(t, c2)
}

func TestMqttPublishNatsSubscribe(t *testing.T) {
	pl := []byte("payload")
	pp := pkg.SimplePublish("testing/s.o.m.e/topic", pl)
	gotIt := make(chan bool, 1)
	nc := full.NatsConnect(t, natsPort)
	defer nc.Close()

	_, err := nc.Subscribe("testing.s/o/m/e.>", func(m *nats.Msg) {
		if !bytes.Equal(pl, m.Data) {
			t.Error("nats subscription did not receive expected data")
		}
		gotIt <- true
	})
	if err != nil {
		t.Fatal(err)
	}

	c2 := full.MqttConnectClean(t, mqttPort)
	full.MqttSend(t, c2, pp)
	full.MqttDisconnect(t, c2)
	full.AssertMessageReceived(t, gotIt)
}

func TestNatsPublishMqttSubscribe(t *testing.T) {
	topic := "testing/some/topic"
	pl := []byte("payload")
	pp := pkg.SimplePublish(topic, pl)

	c1 := full.MqttConnectClean(t, mqttPort)
	sid := nextPacketID()
	full.MqttSend(t, c1, pkg.NewSubscribe(sid, pkg.Topic{Name: "testing/+/topic"}))
	full.MqttExpect(t, c1, pkg.NewSubAck(sid, 0))

	gotIt := make(chan bool, 1)
	go func() {
		full.MqttExpect(t, c1, pp)
		full.MqttDisconnect(t, c1)
		gotIt <- true
	}()

	nc := full.NatsConnect(t, natsPort)
	defer nc.Close()
	err := nc.Publish("testing.some.topic", pl)
	if err != nil {
		t.Fatal(err)
	}
	full.AssertMessageReceived(t, gotIt)
}

func TestNatsPublishMqttSubscribe_qos_1(t *testing.T) {
	topic := "testing/some/topic"
	pl := []byte("payload")

	c1 := full.MqttConnectClean(t, mqttPort)
	sid := nextPacketID()
	full.MqttSend(t, c1, pkg.NewSubscribe(sid, pkg.Topic{Name: topic, QoS: 1}))
	full.MqttExpect(t, c1, pkg.NewSubAck(sid, 1))

	gotIt := make(chan bool, 1)
	go func() {
		full.MqttExpect(t, c1, func(p pkg.Packet) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				full.MqttSend(t, c1, pkg.PubAck(pp.ID()))
				return pp.TopicName() == topic && bytes.Equal(pp.Payload(), pl) && pp.QoSLevel() == 1
			}
			return false
		})
		full.MqttDisconnect(t, c1)
		gotIt <- true
	}()

	nc := full.NatsConnect(t, natsPort)
	defer nc.Close()
	_, err := nc.Request("testing.some.topic", pl, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnubscribe(t *testing.T) {
	topic := "testing/some/topic"
	pp := pkg.SimplePublish(topic, []byte("payload"))
	gotIt := make(chan bool, 1)
	c1 := full.MqttConnectClean(t, mqttPort)
	mid := nextPacketID()
	full.MqttSend(t, c1, pkg.NewSubscribe(mid, pkg.Topic{Name: topic}))
	full.MqttExpect(t, c1, pkg.NewSubAck(mid, 0))
	go func() {
		full.MqttExpect(t, c1, pp)

		uid := nextPacketID()
		full.MqttSend(t, c1, pkg.NewUnsubscribe(uid, topic))
		full.MqttExpect(t, c1, pkg.UnsubAck(uid))
		gotIt <- true
		full.MqttExpectConnClosed(t, c1)
		gotIt <- true
	}()

	c2 := full.MqttConnectClean(t, mqttPort)
	full.MqttSend(t, c2, pp)

	// wait for subscriber to consume and unsubscribe
	full.AssertMessageReceived(t, gotIt)

	// send again, this should not reach subscriber
	full.MqttSend(t, c2, pp)
	full.MqttDisconnect(t, c2)
	full.AssertTimeout(t, gotIt)
	full.MqttDisconnect(t, c1)
}

func TestPublish_qos_2(t *testing.T) {
	conn := full.MqttConnectClean(t, mqttPort)
	full.MqttSend(t, conn, pkg.NewPublish2(
		nextPacketID(), "testing/some/topic", []byte("payload"), 2, false, false))
	full.MqttExpectConnReset(t, conn)
}

func TestPublish_qos_3(t *testing.T) {
	conn := full.MqttConnectClean(t, mqttPort)
	full.MqttSend(t, conn, pkg.NewPublish2(
		nextPacketID(), "testing/some/topic", []byte("payload"), 3, false, false))
	full.MqttExpectConnReset(t, conn)
}
