package test

import (
	"bytes"
	"testing"
	"time"

	"github.com/tada/mqtt-nats/mqtt/pkg"
	"github.com/tada/mqtt-nats/test/full"
)

func TestConnect(t *testing.T) {
	conn := full.MqttConnect(t, mqttPort)
	full.MqttSend(t, conn, pkg.NewConnect(full.NextClientID(), true, 1, nil, nil))
	full.MqttExpect(t, conn, pkg.NewConnAck(false, 0))
	full.MqttDisconnect(t, conn)
}

func TestConnect_sessionPresent(t *testing.T) {
	conn := full.MqttConnect(t, mqttPort)
	c := pkg.NewConnect(full.NextClientID(), false, 1, nil, nil)
	full.MqttSend(t, conn, c)
	full.MqttExpect(t, conn, pkg.NewConnAck(false, 0))
	full.MqttDisconnect(t, conn)

	conn = full.MqttConnect(t, mqttPort)
	full.MqttSend(t, conn, c)
	full.MqttExpect(t, conn, pkg.NewConnAck(true, 0))
	full.MqttDisconnect(t, conn)
}

func TestConnect_will_qos_0(t *testing.T) {
	conn1 := full.MqttConnect(t, mqttPort)
	full.MqttSend(t, conn1, pkg.NewConnect(full.NextClientID(), true, 1, nil, nil))
	full.MqttExpect(t, conn1, pkg.NewConnAck(false, 0))
	mid := full.NextPacketID()
	full.MqttSend(t, conn1, pkg.NewSubscribe(mid, pkg.Topic{Name: "testing/my/will"}))
	full.MqttExpect(t, conn1, pkg.NewSubAck(mid, 0))

	conn2 := full.MqttConnect(t, mqttPort)
	full.MqttSend(t, conn2,
		pkg.NewConnect(full.NextClientID(), true, 1,
			&pkg.Will{
				Topic:   "testing/my/will",
				Message: []byte("the will message")}, nil))
	full.MqttExpect(t, conn2, pkg.NewConnAck(false, 0))
	// forcefully close connection
	_ = conn2.Close()

	full.MqttExpect(t, conn1,
		func(p pkg.Packet) bool {
			pp, ok := p.(*pkg.Publish)
			return ok && pp.TopicName() == "testing/my/will" && pp.QoSLevel() == 0 && !pp.IsDup() && !pp.Retain()
		})
	full.MqttDisconnect(t, conn1)
}

func TestConnect_will_qos_1(t *testing.T) {
	conn := full.MqttConnect(t, mqttPort)
	full.MqttSend(t, conn,
		pkg.NewConnect(full.NextClientID(), true, 1, &pkg.Will{
			Topic:   "testing/my/will",
			Message: []byte("the will message"),
			QoS:     1}, nil))
	full.MqttExpect(t, conn, pkg.NewConnAck(false, 0))
	// forcefully close connection
	_ = conn.Close()

	// Ensure that first package is wasted and a dup is published
	time.Sleep(10 * time.Millisecond)

	conn = full.MqttConnectClean(t, mqttPort)
	mid := full.NextPacketID()
	full.MqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: "testing/my/will", QoS: 1}))
	full.MqttExpect(t, conn,
		pkg.NewSubAck(mid, 1),
		func(p pkg.Packet) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				full.MqttSend(t, conn, pkg.PubAck(pp.ID()))
				return pp.TopicName() == "testing/my/will" && pp.QoSLevel() == 1 && pp.IsDup() && !pp.Retain()
			}
			return false
		})
	full.MqttDisconnect(t, conn)
}

func TestConnect_will_retain_qos_0(t *testing.T) {
	willTopic := "testing/my/will"
	c1 := full.MqttConnect(t, mqttPort)
	full.MqttSend(t, c1,
		pkg.NewConnect(full.NextClientID(), true, 5, &pkg.Will{
			Topic:   willTopic,
			Message: []byte("the will message"),
			Retain:  true}, nil))
	full.MqttExpect(t, c1, pkg.NewConnAck(false, 0))

	// forcefully close connection to make server publish will
	_ = c1.Close()
	time.Sleep(10 * time.Millisecond) // give bridge time to handle retained

	gotIt := make(chan bool, 1)
	go func() {
		c2 := full.MqttConnectClean(t, mqttPort)
		mid := full.NextPacketID()
		full.MqttSend(t, c2, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic}))
		full.MqttExpect(t, c2, pkg.NewSubAck(mid, 0))
		full.MqttExpect(t, c2, func(p pkg.Packet) bool {
			pp, ok := p.(*pkg.Publish)
			return ok && pp.TopicName() == willTopic && pp.QoSLevel() == 0 && !pp.IsDup() && pp.Retain()
		})
		gotIt <- true
		full.MqttDisconnect(t, c2)
	}()

	full.AssertMessageReceived(t, gotIt)

	// check that retained will still exists
	c1 = full.MqttConnectClean(t, mqttPort)
	mid := full.NextPacketID()
	full.MqttSend(t, c1, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic}))
	full.MqttExpect(t, c1,
		pkg.NewSubAck(mid, 0),
		func(p pkg.Packet) bool {
			pp, ok := p.(*pkg.Publish)
			return ok && pp.TopicName() == willTopic && pp.QoSLevel() == 0 && !pp.IsDup() && pp.Retain()
		})

	// drop the retained packet
	full.MqttSend(t, c1, pkg.NewPublish2(0, willTopic, []byte{}, 0, false, true))
	full.MqttDisconnect(t, c1)
}

func TestConnect_will_retain_qos_1(t *testing.T) {
	conn := full.MqttConnect(t, mqttPort)
	willTopic := "testing/my/will"
	willPayload := []byte("the will message")
	full.MqttSend(t, conn,
		pkg.NewConnect(full.NextClientID(), true, 5, &pkg.Will{
			Topic:   willTopic,
			Message: willPayload,
			QoS:     1,
			Retain:  true}, nil))
	full.MqttExpect(t, conn, pkg.NewConnAck(false, 0))
	// forcefully close connection
	_ = conn.Close()

	conn = full.MqttConnectClean(t, mqttPort)
	mid := full.NextPacketID()
	full.MqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic, QoS: 1}))
	var ackID uint16
	full.MqttExpect(t, conn,
		pkg.NewSubAck(mid, 1),
		func(p pkg.Packet) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				ackID = pp.ID()
				return pp.TopicName() == willTopic && bytes.Equal(pp.Payload(), willPayload) && pp.QoSLevel() == 1 && !pp.IsDup() && pp.Retain()
			}
			return false
		})
	full.MqttSend(t, conn, pkg.PubAck(ackID))
	full.MqttDisconnect(t, conn)

	conn = full.MqttConnectClean(t, mqttPort)
	mid = full.NextPacketID()
	full.MqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic, QoS: 1}))
	full.MqttExpect(t, conn,
		pkg.NewSubAck(mid, 1),
		func(p pkg.Packet) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				ackID = pp.ID()
				return pp.TopicName() == willTopic && bytes.Equal(pp.Payload(), willPayload) && pp.QoSLevel() == 1 && !pp.IsDup() && pp.Retain()
			}
			return false
		})
	full.MqttSend(t, conn, pkg.PubAck(ackID))

	// drop the retained packet
	full.MqttSend(t, conn, pkg.NewPublish2(0, willTopic, []byte{}, 1, false, true))
	full.MqttDisconnect(t, conn)
}

func TestConnect_will_retain_qos_1_restart(t *testing.T) {
	conn := full.MqttConnect(t, mqttPort)
	willTopic := "testing/my/will"
	willPayload := []byte("the will message")
	full.MqttSend(t, conn,
		pkg.NewConnect(full.NextClientID(), true, 5, &pkg.Will{
			Topic:   willTopic,
			Message: willPayload,
			QoS:     1,
			Retain:  true}, nil))
	full.MqttExpect(t, conn, pkg.NewConnAck(false, 0))
	// forcefully close connection
	_ = conn.Close()
	time.Sleep(50 * time.Millisecond) // give bridge time to publish will

	conn = full.MqttConnectClean(t, mqttPort)
	mid := full.NextPacketID()
	full.MqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic, QoS: 1}))
	full.MqttExpect(t, conn, pkg.NewSubAck(mid, 1))

	full.MqttExpect(t, conn, func(p pkg.Packet) bool {
		if pp, ok := p.(*pkg.Publish); ok {
			full.MqttSend(t, conn, pkg.PubAck(pp.ID()))
			return pp.TopicName() == willTopic && bytes.Equal(pp.Payload(), willPayload) && pp.QoSLevel() == 1 && !pp.IsDup() && pp.Retain()
		}
		return false
	})
	full.MqttDisconnect(t, conn)

	full.RestartBridge(t, mqttServer)

	conn = full.MqttConnectClean(t, mqttPort)
	mid = full.NextPacketID()
	full.MqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic, QoS: 1}))
	full.MqttExpect(t, conn, pkg.NewSubAck(mid, 1))
	full.MqttExpect(t, conn,
		func(p pkg.Packet) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				full.MqttSend(t, conn, pkg.PubAck(pp.ID()))
				return pp.TopicName() == willTopic && bytes.Equal(pp.Payload(), willPayload) && pp.QoSLevel() == 1 && !pp.IsDup() && pp.Retain()
			}
			return false
		})

	// drop the retained packet
	full.MqttSend(t, conn, pkg.NewPublish2(0, willTopic, []byte{}, 1, false, true))
	full.MqttDisconnect(t, conn)
}

func TestConnect_will_qos_1_restart(t *testing.T) {
	conn := full.MqttConnect(t, mqttPort)
	willTopic := "testing/my/will"
	willPayload := []byte("the will message")
	full.MqttSend(t, conn,
		pkg.NewConnect(full.NextClientID(), true, 5,
			&pkg.Will{
				Topic:   willTopic,
				Message: willPayload,
				QoS:     1},
			&pkg.Credentials{
				User:     "bob",
				Password: []byte("password")}))
	full.MqttExpect(t, conn, pkg.NewConnAck(false, 0))
	// forcefully close connection
	_ = conn.Close()
	time.Sleep(50 * time.Millisecond) // give bridge time to publish will
	full.RestartBridge(t, mqttServer)

	conn = full.MqttConnectClean(t, mqttPort)
	mid := full.NextPacketID()
	full.MqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic, QoS: 1}))
	var ackID uint16
	full.MqttExpect(t, conn,
		pkg.NewSubAck(mid, 1),
		func(p pkg.Packet) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				ackID = pp.ID()
				return pp.TopicName() == willTopic && bytes.Equal(pp.Payload(), willPayload) && pp.QoSLevel() == 1 && pp.IsDup() && !pp.Retain()
			}
			return false
		})
	full.MqttSend(t, conn, pkg.PubAck(ackID))

	// drop the retained packet
	full.MqttSend(t, conn, pkg.NewPublish2(0, willTopic, []byte{}, 1, false, true))
	full.MqttDisconnect(t, conn)
}

func TestPing(t *testing.T) {
	conn := full.MqttConnectClean(t, mqttPort)
	full.MqttSend(t, conn, pkg.PingRequestSingleton)
	full.MqttExpect(t, conn, pkg.PingResponseSingleton)
	full.MqttDisconnect(t, conn)
}

func TestPing_beforeConnect(t *testing.T) {
	conn := full.MqttConnect(t, mqttPort)
	full.MqttSend(t, conn, pkg.PingRequestSingleton)
	full.MqttExpectConnReset(t, conn)
}

func TestConnect_badProtocolVersion(t *testing.T) {
	conn := full.MqttConnect(t, mqttPort)
	cp := pkg.NewConnect(full.NextClientID(), true, 1, nil, nil)
	cp.SetClientLevel(3)
	full.MqttSend(t, conn, cp)
	full.MqttExpect(t, conn, pkg.NewConnAck(false, pkg.RtUnacceptableProtocolVersion))
	full.MqttDisconnect(t, conn)
}

func TestConnect_multiple(t *testing.T) {
	conn := full.MqttConnectClean(t, mqttPort)
	full.MqttSend(t, conn, pkg.NewConnect(full.NextClientID(), true, 1, nil, nil))
	full.MqttExpectConnReset(t, conn)
}

func TestBadPacketLength(t *testing.T) {
	conn := full.MqttConnectClean(t, mqttPort)
	_, err := conn.Write([]byte{0x01, 0xff, 0xff, 0xff, 0xff})
	if err != nil {
		t.Fatal(err)
	}
	full.MqttExpectConnReset(t, conn)
}

func TestBadPacketType(t *testing.T) {
	conn := full.MqttConnectClean(t, mqttPort)
	_, err := conn.Write([]byte{0xff, 0x0})
	if err != nil {
		t.Fatal(err)
	}
	full.MqttExpectConnReset(t, conn)
}
