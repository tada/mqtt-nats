package bridge

import (
	"crypto/tls"

	"github.com/nats-io/nats.go"
)

type Options struct {
	// Path to file where the bridge is persisted. Can be empty if no persistence is desired
	StoragePath string

	// Port is the MQTT port
	Port int

	// NATSUrls is a comma separated list of URLs used when connecting to NATS
	NATSUrls string

	// RepeatRate is the delay in milliseconds between publishing packages that originated in this server
	// that have QoS > 0 but hasn't been acknowledged.
	RepeatRate int

	// RetainedRequestTopic is a NATS topic that a NATS client can publish to after doing a subscribe
	// in order to retrieve any messages that are retained for that subscription. The payload must be
	// the verbatim NATS subscription. Retained messages that matches the subscription will be published
	// to the reply-to topic in the form of a JSON list of objects with a "subject" string and a "payload"
	// base64 encoded string
	RetainedRequestTopic string

	TLSTimeout float64
	TLS        bool
	TLSVerify  bool
	TLSMap     bool
	TLSCert    string
	TLSKey     string
	TLSCaCert  string
	TLSConfig  *tls.Config

	// NATSOpts are options specific to the NATS connection
	NATSOpts []nats.Option
}
