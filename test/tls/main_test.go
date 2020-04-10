package tls

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/nats-io/nats-server/v2/test"

	"github.com/nats-io/nats.go"
	"github.com/tada/mqtt-nats/bridge"
	"github.com/tada/mqtt-nats/logger"
	"github.com/tada/mqtt-nats/test/full"
)

var mqttServer bridge.Bridge

const (
	storageFile          = "mqtt-nats.json"
	mqttPort             = 18883
	natsPort             = 14443
	retainedRequestTopic = "mqtt.retained.request"
)

// tlsDial establishes a tcp tls connection to the given port on the default host
func tlsDial(t *testing.T, port int) net.Conn {
	t.Helper()
	pbs, err := ioutil.ReadFile("testdata/ca.pem")
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(pbs)
	cert, err := tls.LoadX509KeyPair("testdata/client.pem", "testdata/client-key.pem")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := tls.Dial("tcp", ":"+strconv.Itoa(port), &tls.Config{
		ServerName:   "127.0.0.1",
		RootCAs:      roots,
		Certificates: []tls.Certificate{cert},
	})
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestMain(m *testing.M) {
	_ = os.Remove(storageFile)
	natsServer, _ := test.RunServerWithConfig("server.conf")

	natsOpts := []nats.Option{nats.Name("MQTT Bridge")}
	natsOpts = append(natsOpts, nats.ClientCert("testdata/client.pem", "testdata/client-key.pem"))
	natsOpts = append(natsOpts, nats.RootCAs("testdata/ca.pem"))

	// NOTE: Setting level to logger.Debug here is very helpful when authoring and debugging tests but
	//  it also makes the tests very verbose.
	lg := logger.New(logger.Silent, os.Stdout, os.Stderr)

	opts := bridge.Options{
		Port:                 mqttPort,
		NATSUrls:             "localhost:" + strconv.Itoa(natsPort),
		RepeatRate:           50,
		RetainedRequestTopic: retainedRequestTopic,
		StoragePath:          storageFile,
		TLS:                  true,
		TLSCaCert:            "testdata/ca.pem",
		TLSCert:              "testdata/server.pem",
		TLSKey:               "testdata/server-key.pem",
		NATSOpts:             natsOpts}

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
