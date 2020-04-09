// +build citest

package full

import (
	"strconv"
	"testing"

	"github.com/nats-io/nats.go"
)

func NatsConnect(t *testing.T, port int) *nats.Conn {
	nc, err := nats.Connect(":" + strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	return nc
}
