// +build citest

package full

import (
	"io"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/nats-io/nuid"
	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/mqtt/pkg"
	"github.com/tada/mqtt-nats/test/packet"
)

func NextClientID() string {
	return "testclient-" + nuid.Next()
}

// mqttConnect establishes a tcp connection to the given port on the default host
func MqttConnect(t *testing.T, port int) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

// mqttConnectClean establishes a tcp connection to the given port on the default host, sends the
// initial connect packet for a clean session and awaits the CONNACK.
func MqttConnectClean(t *testing.T, port int) net.Conn {
	conn := MqttConnect(t, port)
	MqttSend(t, conn, pkg.NewConnect(NextClientID(), true, 1, nil, nil))
	MqttExpect(t, conn, pkg.NewConnAck(false, 0))
	return conn
}

// mqttDisconnect sends a disconnect packet and closes the connection
func MqttDisconnect(t *testing.T, conn io.WriteCloser) {
	t.Helper()
	defer func() {
		if err := conn.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	buf := mqtt.NewWriter()
	pkg.DisconnectSingleton.Write(buf)
	_, err := conn.Write(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
}

// mqttSend writes the given packets on the given connection
func MqttSend(t *testing.T, conn io.Writer, send ...pkg.Packet) {
	t.Helper()
	buf := mqtt.NewWriter()
	for i := range send {
		send[i].Write(buf)
	}
	_, err := conn.Write(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
}

// mqttExpect will read one packet for each entry in the list of expectations and assert that it is matched
// by that entry. An expectation is either an expected verbatim pkg.Packet or a PacketMatcher function.
func MqttExpect(t *testing.T, conn io.Reader, expectations ...interface{}) {
	t.Helper()
	for _, e := range expectations {
		a := packet.Parse(t, conn)
		switch e := e.(type) {
		case pkg.Packet:
			if !e.Equals(a) {
				t.Fatalf("expected '%s', got '%s'", e, a)
			}
		case func(pkg.Packet) bool:
			if !e(a) {
				t.Fatalf("packet '%s' does not match packet match function", a)
			}
		default:
			t.Fatalf("a %T is not a valid expectation", e)
		}
	}
}

// mqttExpectConnReset will make a read attempt and expect that it fails with an error
func MqttExpectConnReset(t *testing.T, conn net.Conn) {
	t.Helper()
	_, err := conn.Read([]byte{0})
	if err != nil {
		if strings.Contains(err.Error(), "EOF") {
			return
		}
		if strings.Contains(err.Error(), "reset") {
			return
		}
		if strings.Contains(err.Error(), "forcibly closed") {
			return
		}
	}
	t.Fatalf("connection is not reset: %v", err)
}

// mqttExpectConnReset will make a read attempt and expect that it fails with an error
func MqttExpectConnClosed(t *testing.T, conn net.Conn) {
	t.Helper()
	_, err := conn.Read([]byte{0})
	if err != nil && strings.Contains(err.Error(), "closed") {
		return
	}
	t.Fatalf("connection is not closed: %v", err)
}

var packetIDManager = pkg.NewIDManager()

func NextPacketID() uint16 {
	return packetIDManager.NextFreePacketID()
}
