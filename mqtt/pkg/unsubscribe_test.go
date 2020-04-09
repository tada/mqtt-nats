package pkg_test

import (
	"testing"

	"github.com/tada/mqtt-nats/mqtt/pkg"
)

func TestParseUnsubscribe(t *testing.T) {
	writeReadAndCompare(t, pkg.NewUnsubscribe(23, "some/topic", "some/other"), "UNSUBSCRIBE (m23, ['some/topic', 'some/other'])")
}

func TestParseUnsubAck(t *testing.T) {
	writeReadAndCompare(t, pkg.UnsubAck(23), "UNSUBACK (m23)")
}
