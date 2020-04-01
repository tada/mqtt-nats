package bridge

import (
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/tada/mqtt-nats/jsonstream"
	"github.com/tada/mqtt-nats/mqtt/pkg"
	"github.com/tada/mqtt-nats/pio"
)

// natsPub represents a message which originated from this server (such as a client will) that has been
// published to NATS using some given credentials and now awaits a reply.
type natsPub struct {
	// the package that was published
	pp *pkg.Publish

	// user from client connection
	user *string

	// password from client connection
	password []byte
}

func (n *natsPub) MarshalToJSON(w io.Writer) {
	pio.WriteString(`{"pkg":`, w)
	n.pp.MarshalToJSON(w)
	if n.user != nil {
		pio.WriteString(`,"user":`, w)
		jsonstream.WriteString(*n.user, w)
	}
	if n.password != nil {
		pio.WriteString(`,"pw":`, w)
		jsonstream.WriteString(base64.StdEncoding.EncodeToString(n.password), w)
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
		case "pkg":
			n.pp = &pkg.Publish{}
			jsonstream.AssertConsumer(js, n.pp)
		case "user":
			u := jsonstream.AssertString(js)
			n.user = &u
		case "pw":
			p, err := base64.StdEncoding.DecodeString(jsonstream.AssertString(js))
			if err != nil {
				panic(pio.Error{Cause: err})
			}
			n.password = p
		}
	}
}
