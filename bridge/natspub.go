package bridge

import (
	"encoding/json"
	"io"

	"github.com/tada/catch/pio"
	"github.com/tada/jsonstream"
	"github.com/tada/mqtt-nats/mqtt/pkg"
)

// natsPub represents a message which originated from this server (such as a client will) that has been
// published to NATS using some given credentials and now awaits a reply.
type natsPub struct {
	// pp is the packet that was published
	pp *pkg.Publish

	// creds are the client credentials for the publication
	creds *pkg.Credentials
}

func (n *natsPub) MarshalToJSON(w io.Writer) {
	pio.WriteString(w, `{"m":`)
	n.pp.MarshalToJSON(w)
	if n.creds != nil {
		pio.WriteString(w, `,"c":`)
		n.creds.MarshalToJSON(w)
	}
	pio.WriteByte(w, '}')
}

func (n *natsPub) UnmarshalFromJSON(js jsonstream.Decoder, t json.Token) {
	jsonstream.AssertDelim(t, '{')
	for {
		k, ok := js.ReadStringOrEnd('}')
		if !ok {
			break
		}
		switch k {
		case "m":
			n.pp = &pkg.Publish{}
			if !js.ReadConsumer(n.pp) {
				n.pp = nil
			}
		case "c":
			n.creds = &pkg.Credentials{}
			if !js.ReadConsumer(n.creds) {
				n.creds = nil
			}
		}
	}
}
