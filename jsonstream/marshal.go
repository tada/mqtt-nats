// Package jsonstream contains a Streamer and a Consumer interface and utility functions to make true streaming
// easy to implement.
package jsonstream

import (
	"bytes"
	"io"
	"strings"

	"github.com/tada/mqtt-nats/pio"
)

// Streamer is an interface that can be implemented by instances that stream output.
type Streamer interface {
	// MarshalToJSON serializes the instance onto the given writer using the pio.Write
	// methods. It must be guarded by pio.Catch to recover errors.
	//
	// See top level function MarshalJSON(JSONStreamer) for more info
	MarshalToJSON(io.Writer)
}

// Marshal is called by instances that implement both the standard JSONMarshaller interface
// and the Streamer interface. The function creates a bytes.Buffer onto which the json stream
// is written. The resulting bytes are returned.
//
// The function will recover a pio.Error panic and return its cause.
func Marshal(s Streamer) (result []byte, err error) {
	err = pio.Catch(func() error {
		w := bytes.Buffer{}
		s.MarshalToJSON(&w)
		result = w.Bytes()
		return nil
	})
	return
}

// WriteString writes s as double quoted string on the writer using '\' to escape
// the '"' and the '\'.
//
// If an error occurs the method panics with a Error with the Cause set to that error
func WriteString(s string, w io.Writer) {
	pio.WriteByte('"', w)
	r := strings.NewReader(s)
	for {
		r, _, err := r.ReadRune()
		if err == io.EOF {
			pio.WriteByte('"', w)
			return
		}
		switch r {
		case '"', '\\':
			pio.WriteByte('\\', w)
			pio.WriteByte(byte(r), w)
		default:
			pio.WriteRune(r, w)
		}
	}
}
