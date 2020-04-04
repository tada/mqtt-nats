package pkg

import "testing"

func TestParseSubscribe(t *testing.T) {
	writeReadAndCompare(t, NewSubscribe(23, Topic{Name: "some/topic", QoS: 1}),
		ParseSubscribe, "SUBSCRIBE (m23, q1, 'some/topic')")
	writeReadAndCompare(t, NewSubscribe(23, Topic{Name: "some/topic", QoS: 0}, Topic{Name: "some/other"}),
		ParseSubscribe, "SUBSCRIBE (m23, [(q0, 'some/topic'), (q0, 'some/other')])")
}
