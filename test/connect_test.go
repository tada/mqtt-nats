package test

import (
	"bytes"
	"testing"

	"github.com/nats-io/nuid"
	"github.com/tada/mqtt-nats/mqtt/pkg"
)

func TestConnect(t *testing.T) {
	conn := mqttConnect(t, mqttPort)
	mqttSend(t, conn, pkg.NewConnect("testclient-"+nuid.Next(), true, 1, nil, "", nil))
	mqttExpect(t, conn, pkg.NewAckConnect(false, 0))
	mqttDisconnect(t, conn)
}

func TestConnect_sessionPresent(t *testing.T) {
	conn := mqttConnect(t, mqttPort)
	c := pkg.NewConnect("testclient-"+nuid.Next(), false, 1, nil, "", nil)
	mqttSend(t, conn, c)
	mqttExpect(t, conn, pkg.NewAckConnect(false, 0))
	mqttDisconnect(t, conn)

	conn = mqttConnect(t, mqttPort)
	mqttSend(t, conn, c)
	mqttExpect(t, conn, pkg.NewAckConnect(true, 0))
	mqttDisconnect(t, conn)
}

func TestConnect_will_qos_0(t *testing.T) {
	conn1 := mqttConnect(t, mqttPort)
	mqttSend(t, conn1, pkg.NewConnect("testclient-"+nuid.Next(), true, 1, nil, "", nil))
	mqttExpect(t, conn1, pkg.NewAckConnect(false, 0))
	mid := nextPackageID()
	mqttSend(t, conn1, pkg.NewSubscribe(mid, pkg.Topic{Name: "testing/my/will"}))
	mqttExpect(t, conn1, pkg.NewSubAck(mid, 0))

	conn2 := mqttConnect(t, mqttPort)
	mqttSend(t, conn2,
		pkg.NewConnect("testclient-"+nuid.Next(), true, 1,
			&pkg.Will{
				Topic:   "testing/my/will",
				Message: []byte("the will message")}, "", nil))
	mqttExpect(t, conn2, pkg.NewAckConnect(false, 0))
	// forcefully close connection
	_ = conn2.Close()

	mqttExpect(t, conn1,
		func(p pkg.Package) bool {
			pp, ok := p.(*pkg.Publish)
			return ok && pp.TopicName() == "testing/my/will" && pp.QoSLevel() == 0 && !pp.IsDup() && !pp.Retain()
		})
	mqttDisconnect(t, conn1)
}

func TestConnect_will_qos_1(t *testing.T) {
	conn := mqttConnect(t, mqttPort)
	mqttSend(t, conn,
		pkg.NewConnect("testclient-"+nuid.Next(), true, 1, &pkg.Will{
			Topic:   "testing/my/will",
			Message: []byte("the will message"),
			QoS:     1}, "", nil))
	mqttExpect(t, conn, pkg.NewAckConnect(false, 0))
	// forcefully close connection
	_ = conn.Close()

	conn = mqttConnectClean(t, mqttPort)
	mid := nextPackageID()
	mqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: "testing/my/will", QoS: 1}))
	mqttExpect(t, conn,
		pkg.NewSubAck(mid, 1),
		func(p pkg.Package) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				mqttSend(t, conn, pkg.PubAck(pp.ID()))
				return pp.TopicName() == "testing/my/will" && pp.QoSLevel() == 1 && pp.IsDup() && !pp.Retain()
			}
			return false
		})
	mqttDisconnect(t, conn)
}

func TestConnect_will_retain_qos_0(t *testing.T) {
	willTopic := "testing/my/will"
	c1 := mqttConnect(t, mqttPort)
	mqttSend(t, c1,
		pkg.NewConnect("testclient-"+nuid.Next(), true, 5, &pkg.Will{
			Topic:   willTopic,
			Message: []byte("the will message"),
			Retain:  true}, "", nil))
	mqttExpect(t, c1, pkg.NewAckConnect(false, 0))

	gotIt := make(chan bool, 1)
	go func() {
		c2 := mqttConnectClean(t, mqttPort)
		mid := nextPackageID()
		mqttSend(t, c2, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic}))
		mqttExpect(t, c2,
			pkg.NewSubAck(mid, 0),
			func(p pkg.Package) bool {
				pp, ok := p.(*pkg.Publish)
				return ok && pp.TopicName() == willTopic && pp.QoSLevel() == 0 && !pp.IsDup() && pp.Retain()
			})
		gotIt <- true
		mqttDisconnect(t, c2)
	}()

	// forcefully close connection to make server publish will
	_ = c1.Close()
	assertMessageReceived(t, gotIt)

	// check that retained will still exists
	c1 = mqttConnectClean(t, mqttPort)
	mid := nextPackageID()
	mqttSend(t, c1, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic}))
	mqttExpect(t, c1,
		pkg.NewSubAck(mid, 0),
		func(p pkg.Package) bool {
			pp, ok := p.(*pkg.Publish)
			return ok && pp.TopicName() == willTopic && pp.QoSLevel() == 0 && !pp.IsDup() && pp.Retain()
		})

	// drop the retained package
	mqttSend(t, c1, pkg.NewPublish2(0, willTopic, []byte{}, 0, false, true))
	mqttDisconnect(t, c1)
}

func TestConnect_will_retain_qos_1(t *testing.T) {
	conn := mqttConnect(t, mqttPort)
	willTopic := "testing/my/will"
	willPayload := []byte("the will message")
	mqttSend(t, conn,
		pkg.NewConnect("testclient-"+nuid.Next(), true, 5, &pkg.Will{
			Topic:   willTopic,
			Message: willPayload,
			QoS:     1,
			Retain:  true}, "", nil))
	mqttExpect(t, conn, pkg.NewAckConnect(false, 0))
	// forcefully close connection
	_ = conn.Close()

	conn = mqttConnectClean(t, mqttPort)
	mid := nextPackageID()
	mqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic, QoS: 1}))
	var ackID uint16
	mqttExpect(t, conn,
		pkg.NewSubAck(mid, 1),
		func(p pkg.Package) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				ackID = pp.ID()
				return pp.TopicName() == willTopic && bytes.Equal(pp.Payload(), willPayload) && pp.QoSLevel() == 1 && !pp.IsDup() && pp.Retain()
			}
			return false
		})
	mqttSend(t, conn, pkg.PubAck(ackID))
	mqttDisconnect(t, conn)

	conn = mqttConnectClean(t, mqttPort)
	mid = nextPackageID()
	mqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic, QoS: 1}))
	mqttExpect(t, conn,
		pkg.NewSubAck(mid, 1),
		func(p pkg.Package) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				ackID = pp.ID()
				return pp.TopicName() == willTopic && bytes.Equal(pp.Payload(), willPayload) && pp.QoSLevel() == 1 && !pp.IsDup() && pp.Retain()
			}
			return false
		})
	mqttSend(t, conn, pkg.PubAck(ackID))

	// drop the retained package
	mqttSend(t, conn, pkg.NewPublish2(0, willTopic, []byte{}, 1, false, true))
	mqttDisconnect(t, conn)
}

func TestConnect_will_retain_qos_1_restart(t *testing.T) {
	conn := mqttConnect(t, mqttPort)
	willTopic := "testing/my/will"
	willPayload := []byte("the will message")
	mqttSend(t, conn,
		pkg.NewConnect("testclient-"+nuid.Next(), true, 5, &pkg.Will{
			Topic:   willTopic,
			Message: willPayload,
			QoS:     1,
			Retain:  true}, "", nil))
	mqttExpect(t, conn, pkg.NewAckConnect(false, 0))
	// forcefully close connection
	_ = conn.Close()

	conn = mqttConnectClean(t, mqttPort)
	mid := nextPackageID()
	mqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic, QoS: 1}))
	var ackID uint16
	mqttExpect(t, conn,
		pkg.NewSubAck(mid, 1),
		func(p pkg.Package) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				ackID = pp.ID()
				return pp.TopicName() == willTopic && bytes.Equal(pp.Payload(), willPayload) && pp.QoSLevel() == 1 && !pp.IsDup() && pp.Retain()
			}
			return false
		})
	mqttSend(t, conn, pkg.PubAck(ackID))
	mqttDisconnect(t, conn)

	ready := make(chan bool, 1)
	go func() {
		if err := mqttServer.Restart(ready); err != nil {
			t.Fatal(err)
		}
	}()

	if !<-ready {
		return
	}

	conn = mqttConnectClean(t, mqttPort)
	mid = nextPackageID()
	mqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic, QoS: 1}))
	mqttExpect(t, conn,
		pkg.NewSubAck(mid, 1),
		func(p pkg.Package) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				ackID = pp.ID()
				return pp.TopicName() == willTopic && bytes.Equal(pp.Payload(), willPayload) && pp.QoSLevel() == 1 && !pp.IsDup() && pp.Retain()
			}
			return false
		})
	mqttSend(t, conn, pkg.PubAck(ackID))

	// drop the retained package
	mqttSend(t, conn, pkg.NewPublish2(0, willTopic, []byte{}, 1, false, true))
	mqttDisconnect(t, conn)
}

func TestConnect_will_qos_1_restart(t *testing.T) {
	conn := mqttConnect(t, mqttPort)
	willTopic := "testing/my/will"
	willPayload := []byte("the will message")
	mqttSend(t, conn,
		pkg.NewConnect("testclient-"+nuid.Next(), true, 5, &pkg.Will{
			Topic:   willTopic,
			Message: willPayload,
			QoS:     1}, "bob", []byte("password")))
	mqttExpect(t, conn, pkg.NewAckConnect(false, 0))
	// forcefully close connection
	_ = conn.Close()

	ready := make(chan bool, 1)
	go func() {
		if err := mqttServer.Restart(ready); err != nil {
			t.Fatal(err)
		}
	}()

	if !<-ready {
		return
	}

	conn = mqttConnectClean(t, mqttPort)
	mid := nextPackageID()
	mqttSend(t, conn, pkg.NewSubscribe(mid, pkg.Topic{Name: willTopic, QoS: 1}))
	var ackID uint16
	mqttExpect(t, conn,
		pkg.NewSubAck(mid, 1),
		func(p pkg.Package) bool {
			if pp, ok := p.(*pkg.Publish); ok {
				ackID = pp.ID()
				return pp.TopicName() == willTopic && bytes.Equal(pp.Payload(), willPayload) && pp.QoSLevel() == 1 && pp.IsDup() && !pp.Retain()
			}
			return false
		})
	mqttSend(t, conn, pkg.PubAck(ackID))

	// drop the retained package
	mqttSend(t, conn, pkg.NewPublish2(0, willTopic, []byte{}, 1, false, true))
	mqttDisconnect(t, conn)
}

func TestPing(t *testing.T) {
	conn := mqttConnectClean(t, mqttPort)
	mqttSend(t, conn, pkg.PingRequestSingleton)
	mqttExpect(t, conn, pkg.PingResponseSingleton)
	mqttDisconnect(t, conn)
}

func TestPing_beforeConnect(t *testing.T) {
	conn := mqttConnect(t, mqttPort)
	mqttSend(t, conn, pkg.PingRequestSingleton)
	mqttExpectConnReset(t, conn)
}

func TestConnect_badProtocolVersion(t *testing.T) {
	conn := mqttConnect(t, mqttPort)
	cp := pkg.NewConnect("testclient-"+nuid.Next(), true, 1, nil, "", nil)
	cp.SetClientLevel(3)
	mqttSend(t, conn, cp)
	mqttExpect(t, conn, pkg.NewAckConnect(false, pkg.RtUnacceptableProtocolVersion))
	mqttDisconnect(t, conn)
}

func TestConnect_multiple(t *testing.T) {
	conn := mqttConnectClean(t, mqttPort)
	mqttSend(t, conn, pkg.NewConnect("testclient-"+nuid.Next(), true, 1, nil, "", nil))
	mqttExpectConnReset(t, conn)
}
