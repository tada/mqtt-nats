package pkg

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/tada/mqtt-nats/jsonstream"
	"github.com/tada/mqtt-nats/pio"
)

type Credentials struct {
	User     string
	Password []byte
}

func (c *Credentials) MarshalToJSON(w io.Writer) {
	pio.WriteByte('{', w)
	if c.User != "" {
		pio.WriteString(`"u":`, w)
		jsonstream.WriteString(c.User, w)
	}
	if c.Password != nil {
		if c.User != "" {
			pio.WriteByte(',', w)
		}
		pio.WriteString(`"p":`, w)
		jsonstream.WriteString(base64.StdEncoding.EncodeToString(c.Password), w)
	}
	pio.WriteByte('}', w)
}

func (c *Credentials) UnmarshalFromJSON(js *json.Decoder, t json.Token) {
	jsonstream.AssertDelimToken(t, '{')
	for {
		k, ok := jsonstream.AssertStringOrEnd(js, '}')
		if !ok {
			break
		}
		switch k {
		case "u":
			c.User = jsonstream.AssertString(js)
		case "p":
			p, err := base64.StdEncoding.DecodeString(jsonstream.AssertString(js))
			if err != nil {
				panic(pio.Error{Cause: err})
			}
			c.Password = p
		}
	}
}

// Equals returns true if this instance is equal to the given instance, false if not
func (c *Credentials) Equals(oc *Credentials) bool {
	return c.User == oc.User && bytes.Equal(c.Password, oc.Password)
}
