package jsonstream

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/tada/mqtt-nats/pio"
)

func decoderOn(s string) *json.Decoder {
	js := json.NewDecoder(bytes.NewReader([]byte(s)))
	js.UseNumber()
	return js
}

func TestAssertDelim(t *testing.T) {
	js := decoderOn("{}")
	err := pio.Catch(func() error {
		AssertDelim(js, '{')
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = pio.Catch(func() error {
		AssertDelim(js, '{')
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	err = pio.Catch(func() error {
		AssertDelim(js, '{')
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssertString(t *testing.T) {
	js := decoderOn(`"a"`)
	err := pio.Catch(func() (e error) {
		if s := AssertString(js); s != "a" {
			e = fmt.Errorf(`expected "a", got "%s"`, s)
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	js = decoderOn(`1`)
	err = pio.Catch(func() error {
		AssertString(js)
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssertStringOrEnd(t *testing.T) {
	js := decoderOn(`["a"]`)
	err := pio.Catch(func() (e error) {
		AssertDelim(js, '[')
		if _, ok := AssertStringOrEnd(js, ']'); !ok {
			e = errors.New(`expected "a", got end`)
		}
		if s, ok := AssertStringOrEnd(js, ']'); ok {
			e = fmt.Errorf(`expected end, got "%s"`, s)
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	err = pio.Catch(func() error {
		_, _ = AssertStringOrEnd(js, ']')
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	js = decoderOn(`23`)
	err = pio.Catch(func() error {
		_, _ = AssertStringOrEnd(js, ']')
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssertInt(t *testing.T) {
	js := decoderOn(`1`)
	err := pio.Catch(func() (e error) {
		if s := AssertInt(js); s != 1 {
			e = fmt.Errorf(`expected 1, got %d`, s)
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	js = decoderOn(`"a"`)
	err = pio.Catch(func() error {
		AssertInt(js)
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssertIntOrEnd(t *testing.T) {
	js := decoderOn(`[42]`)
	err := pio.Catch(func() (e error) {
		AssertDelim(js, '[')
		if _, ok := AssertIntOrEnd(js, ']'); !ok {
			e = errors.New(`expected "a", got end`)
		}
		if i, ok := AssertIntOrEnd(js, ']'); ok {
			e = fmt.Errorf(`expected end, got "%d"`, i)
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	err = pio.Catch(func() error {
		_, _ = AssertIntOrEnd(js, ']')
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	js = decoderOn(`"a"`)
	err = pio.Catch(func() error {
		_, _ = AssertIntOrEnd(js, ']')
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

type testConsumer struct {
	t *testing.T
	m string
	i int64
}

func (t *testConsumer) UnmarshalFromJSON(js *json.Decoder, firstToken json.Token) {
	AssertDelimToken(firstToken, '{')
	var s string
	ok := true
	for ok {
		if s, ok = AssertStringOrEnd(js, '}'); ok {
			switch s {
			case "m":
				t.m = AssertString(js)
			case "i":
				t.i = AssertInt(js)
			default:
				t.t.Fatalf("unexpected string %q", s)
			}
		}
	}
}

func TestAssertConsumer(t *testing.T) {
	js := decoderOn(`{"m":"message","i":42}`)
	err := pio.Catch(func() (e error) {
		tc := &testConsumer{t: t}
		AssertConsumer(js, tc)
		if !(tc.m == "message" && tc.i == 42) {
			t.Fatal("unexpected consumer values")
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	err = pio.Catch(func() error {
		tc := &testConsumer{t: t}
		AssertConsumer(js, tc)
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssertConsumerOrEnd(t *testing.T) {
	js := decoderOn(`[{"m":"message","i":42}]`)
	err := pio.Catch(func() (e error) {
		AssertDelim(js, '[')
		tc := &testConsumer{t: t}
		if ok := AssertConsumerOrEnd(js, tc, ']'); !ok {
			e = errors.New(`expected consumer, got end`)
		}
		if ok := AssertConsumerOrEnd(js, tc, ']'); ok {
			e = fmt.Errorf(`expected end, got consumer %v`, tc)
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	err = pio.Catch(func() error {
		tc := &testConsumer{t: t}
		_ = AssertConsumerOrEnd(js, tc, ']')
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
