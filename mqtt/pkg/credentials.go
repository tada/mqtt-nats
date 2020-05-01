package pkg

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/tada/catch"

	"github.com/tada/catch/pio"
	"github.com/tada/jsonstream"
)

// Credentials are user credentials that originates from an MQTT CONNECT packet.
type Credentials struct {
	User     string
	Password []byte
}

// MarshalToJSON streams the JSON encoded form of this instance onto the given io.Writer
func (c *Credentials) MarshalToJSON(w io.Writer) {
	pio.WriteByte(w, '{')
	if c.User != "" {
		pio.WriteString(w, `"u":`)
		jsonstream.WriteString(w, c.User)
	}
	if c.Password != nil {
		if c.User != "" {
			pio.WriteByte(w, ',')
		}
		pio.WriteString(w, `"p":`)
		jsonstream.WriteString(w, base64.StdEncoding.EncodeToString(c.Password))
	}
	pio.WriteByte(w, '}')
}

// UnmarshalFromJSON initializes this instance from the tokens stream provided by the json.Decoder. The
// first token has already been read and is passed as an argument.
func (c *Credentials) UnmarshalFromJSON(js jsonstream.Decoder, t json.Token) {
	jsonstream.AssertDelim(t, '{')
	for {
		k, ok := js.ReadStringOrEnd('}')
		if !ok {
			break
		}
		switch k {
		case "u":
			c.User = js.ReadString()
		case "p":
			p, err := base64.StdEncoding.DecodeString(js.ReadString())
			if err != nil {
				panic(catch.Error(err))
			}
			c.Password = p
		}
	}
}

// Equals returns true if this instance is equal to the given instance, false if not
func (c *Credentials) Equals(oc *Credentials) bool {
	return c.User == oc.User && bytes.Equal(c.Password, oc.Password)
}
