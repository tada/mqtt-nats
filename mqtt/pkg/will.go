package pkg

import (
	"bytes"
	"fmt"
)

// Will is the optional client will in the MQTT connect packet
type Will struct {
	Topic   string
	Message []byte
	QoS     byte
	Retain  bool
}

// Equals returns true if this instance is equal to the given instance, false if not
func (w *Will) Equals(ow *Will) bool {
	return w.Retain == ow.Retain && w.QoS == ow.QoS && w.Topic == ow.Topic && bytes.Equal(w.Message, ow.Message)
}

// String returns a brief string representation of the will. Suitable for logging
func (w *Will) String() string {
	r := 0
	if w.Retain {
		r = 1
	}
	return fmt.Sprintf("w(r%d, q%d, '%s', ... (%d bytes))", r, w.QoS, w.Topic, len(w.Message))
}
