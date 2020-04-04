package pkg

import "testing"

func TestParseUnsubscribe(t *testing.T) {
	writeReadAndCompare(t, NewUnsubscribe(23, "some/topic", "some/other"), ParseUnsubscribe, "UNSUBSCRIBE (m23, ['some/topic', 'some/other'])")
}
