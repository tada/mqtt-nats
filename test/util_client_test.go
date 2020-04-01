package test

import (
	"io"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/nats-io/nuid"
	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/mqtt/pkg"
)

// mqttConnect establishes a tcp connection to the given port on the default host
func mqttConnect(t *testing.T, port int) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

// mqttConnect establishes a tcp connection to the given port on the default host, sends the
// initial connect package for a clean session and awaits the CONNACK.
func mqttConnectClean(t *testing.T, port int) net.Conn {
	conn := mqttConnect(t, port)
	mqttSend(t, conn, pkg.NewConnect("testclient-"+nuid.Next(), true, 1, nil, "", nil))
	mqttExpect(t, conn, pkg.NewAckConnect(false, 0))
	return conn
}

// mqttDisconnect sends a disconnect package and closes the connection
func mqttDisconnect(t *testing.T, conn io.WriteCloser) {
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

// mqttSend writes the given packages on the given connection
func mqttSend(t *testing.T, conn io.Writer, send ...pkg.Package) {
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

// mqttExpect will read one package for each entry in the list of expectations and assert that it is matched
// by that entry. An expectation is either an expected verbatim pkg.Package or a PackageMatcher function.
func mqttExpect(t *testing.T, conn io.Reader, expectations ...interface{}) {
	t.Helper()
	for _, e := range expectations {
		a := parsePackage(t, conn)
		switch e := e.(type) {
		case pkg.Package:
			if !e.Equals(a) {
				t.Fatalf("expected '%s', got '%s'", e, a)
			}
		case func(pkg.Package) bool:
			if !e(a) {
				t.Fatalf("package '%s' does not match PackageMatcher", a)
			}
		default:
			t.Fatalf("a %T is not a valid expectation", e)
		}
	}
}

// mqttExpectConnReset will make a read attempt and expect that it fails with an error
func mqttExpectConnReset(t *testing.T, conn net.Conn) {
	t.Helper()
	_, err := conn.Read([]byte{0})
	if err != nil {
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
func mqttExpectConnClosed(t *testing.T, conn net.Conn) {
	t.Helper()
	_, err := conn.Read([]byte{0})
	if err != nil && strings.Contains(err.Error(), "closed") {
		return
	}
	t.Fatalf("connection is not closed: %v", err)
}

var packageIDManager = pkg.NewIDManager()

func nextPackageID() uint16 {
	return packageIDManager.NextFreePackageID()
}
