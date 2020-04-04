package bridge

import (
	"encoding/json"
	"io"

	"github.com/tada/mqtt-nats/jsonstream"
	"github.com/tada/mqtt-nats/mqtt/pkg"
	"github.com/tada/mqtt-nats/pio"
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
	pio.WriteString(`{"m":`, w)
	n.pp.MarshalToJSON(w)
	if n.creds != nil {
		pio.WriteString(`,"c":`, w)
		n.creds.MarshalToJSON(w)
	}
	pio.WriteByte('}', w)
}

func (n *natsPub) UnmarshalFromJSON(js *json.Decoder, t json.Token) {
	jsonstream.AssertDelimToken(t, '{')
	for {
		k, ok := jsonstream.AssertStringOrEnd(js, '}')
		if !ok {
			break
		}
		switch k {
		case "m":
			n.pp = &pkg.Publish{}
			jsonstream.AssertConsumer(js, n.pp)
		case "c":
			n.creds = &pkg.Credentials{}
			jsonstream.AssertConsumer(js, n.creds)
		}
	}
}
