package jsonstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/tada/mqtt-nats/pio"
)

// A Consumer is a value that can initialize itself from a json.Decoder
type Consumer interface {
	// Initialize this instance from a json.Decoder and one token that has
	// already been read from that decoder.
	UnmarshalFromJSON(js *json.Decoder, firstToken json.Token)
}

// unexpectedError converts io.EOF to io.ErrUnexpectedEOF. The argument is returned for all other values
func unexpectedError(err error) error {
	if err == io.EOF {
		err = &pio.Error{Cause: io.ErrUnexpectedEOF}
	}
	return err
}

// Unmarshal initializes the given Consumer from a sequence of bytes that must be valid JSON.
func Unmarshal(c Consumer, bs []byte) error {
	return pio.Catch(func() error {
		js := json.NewDecoder(bytes.NewReader(bs))
		js.UseNumber()
		AssertConsumer(js, c)
		return nil
	})
}

// AssertDelim reads next token from the decoder and asserts that it is equal to the given delimiter. A panic
// with a pio.Error is raised if that is not the case.
func AssertDelim(js *json.Decoder, delim byte) {
	t, err := js.Token()
	if err == nil {
		AssertDelimToken(t, delim)
		return
	}
	panic(unexpectedError(err))
}

// AssertDelimToken asserts that the given token is equal to the given delimiter. A panic
// with a pio.Error is raised if that is not the case.
func AssertDelimToken(t json.Token, delim byte) {
	if d, ok := t.(json.Delim); ok {
		s := d.String()
		if len(s) == 1 && s[0] == delim {
			return
		}
	}
	panic(&pio.Error{Cause: fmt.Errorf("expected delimiter '%c', got %T %v", delim, t, t)})
}

// AssertString reads next token from the decoder and asserts that it is a string. The function returns the string
// or raises a panic with a pio.Error if an error occurred or if the token didn't match a string.
func AssertString(js *json.Decoder) string {
	t, err := js.Token()
	if err == nil {
		if s, ok := t.(string); ok {
			return s
		}
		err = &pio.Error{Cause: fmt.Errorf("expected a string, got %T %v", t, t)}
	}
	panic(unexpectedError(err))
}

// AssertInt reads next token from the decoder and asserts that it is an integer. The function returns the integer
// or raises a panic with a pio.Error if an error occurred or if the token didn't match a string.
//
// The decoder must be configured with UseNumber()
func AssertInt(js *json.Decoder) int64 {
	t, err := js.Token()
	if err == nil {
		if s, ok := t.(json.Number); ok {
			var i int64
			if i, err = s.Int64(); err == nil {
				return i
			}
		}
		err = &pio.Error{Cause: fmt.Errorf("expected an integer, got %T %v", t, t)}
	}
	panic(unexpectedError(err))
}

// AssertStringOrEnd reads next token from the decoder and asserts that it is either a string or a delimiter that
// matches the given end. The function returns the string and true if a string is found or an empty string and
// false if the delimiter was found. A panic with a pio.Error is raised if neither of those cases are true.
func AssertStringOrEnd(js *json.Decoder, end byte) (string, bool) {
	t, err := js.Token()
	if err == nil {
		switch t := t.(type) {
		case string:
			return t, true
		case json.Delim:
			s := t.String()
			if len(s) == 1 && s[0] == end {
				return ``, false
			}
		}
		err = &pio.Error{Cause: fmt.Errorf("expected a string or the delimiter '%c' got %T %v", end, t, t)}
	}
	panic(unexpectedError(err))
}

// AssertIntOrEnd reads next token from the decoder and asserts that it is either an integer or a delimiter that
// matches the given end. The function returns the integer and true if an integer is found or 0 and false
// if the delimiter was found. A panic with a pio.Error is raised if neither of those cases are true.
//
// The decoder must be configured with UseNumber()
func AssertIntOrEnd(js *json.Decoder, end byte) (int64, bool) {
	t, err := js.Token()
	if err == nil {
		switch t := t.(type) {
		case json.Number:
			var i int64
			if i, err = t.Int64(); err == nil {
				return i, true
			}
		case json.Delim:
			s := t.String()
			if len(s) == 1 && s[0] == end {
				return 0, false
			}
		}
		err = &pio.Error{Cause: fmt.Errorf("expected an integer or the delimiter '%c' got %T %v", end, t, t)}
	}
	panic(unexpectedError(err))
}

// AssertConsumer reads next token from the decoder and passes that token to the given consumers UnmarshalFromJSON.
func AssertConsumer(js *json.Decoder, c Consumer) {
	t, err := js.Token()
	if err == nil {
		c.UnmarshalFromJSON(js, t)
		return
	}
	panic(unexpectedError(err))
}

// AssertConsumerOrEnd reads next token from the decoder and asserts that it is either an consumer or a delimiter that
// matches the given end. The function returns true if a consumer is found and false if the delimiter was found. A
// panic with a pio.Error is raised if neither of those cases are true.
func AssertConsumerOrEnd(js *json.Decoder, c Consumer, end byte) bool {
	t, err := js.Token()
	if err == nil {
		if d, ok := t.(json.Delim); ok {
			s := d.String()
			if len(s) == 1 && s[0] == end {
				return false
			}
		}
		c.UnmarshalFromJSON(js, t)
		return true
	}
	panic(unexpectedError(err))
}
