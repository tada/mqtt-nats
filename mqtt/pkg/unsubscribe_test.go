package pkg

import "testing"

func TestParseUnsubscribe(t *testing.T) {
	writeReadAndCompare(t, NewUnsubscribe(23, "some/topic", "some/other"), ParseUnsubscribe, "UNSUBSCRIBE (m23, ['some/topic', 'some/other'])")
}

func TestParseUnsubAck(t *testing.T) {
	writeReadAndCompare(t, UnsubAck(23), ParseUnsubAck, "UNSUBACK (m23)")
}
