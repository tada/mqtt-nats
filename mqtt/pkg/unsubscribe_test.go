package pkg

import "testing"

func TestParseUnsubscribe(t *testing.T) {
	writeReadAndCompare(t, NewUnsubscribe(23, "some/topic", "some/other"), "UNSUBSCRIBE (m23, ['some/topic', 'some/other'])")
}

func TestParseUnsubAck(t *testing.T) {
	writeReadAndCompare(t, UnsubAck(23), "UNSUBACK (m23)")
}
