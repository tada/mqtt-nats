package test

import (
	"strconv"
	"testing"

	"github.com/nats-io/nats.go"
)

func natsConnect(t *testing.T, port int) *nats.Conn {
	nc, err := nats.Connect(":" + strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	return nc
}
