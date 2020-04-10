// +build citest

package full

import (
	"strconv"
	"testing"

	"github.com/nats-io/nats.go"
)

// NatsConnect creates a new NATS connection on the given port.
func NatsConnect(t *testing.T, port int) *nats.Conn {
	nc, err := nats.Connect(":" + strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	return nc
}
