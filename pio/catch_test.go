package pio_test

import (
	"errors"
	"testing"

	"github.com/tada/mqtt-nats/pio"
)

func TestCatch_normal(t *testing.T) {
	e := errors.New("the error")
	err := pio.Catch(func() error {
		panic(&pio.Error{e})
	})
	if err != e {
		t.Errorf("expected %v, got %v", e, err)
	}
}

func TestCatch_normalByValue(t *testing.T) {
	e := errors.New("the error")
	err := pio.Catch(func() error {
		panic(pio.Error{e})
	})
	if err != e {
		t.Errorf("expected %v, got %v", e, err)
	}
}

func TestCatch_other(t *testing.T) {
	e := errors.New("the error")
	defer func() {
		err := recover()
		if err != e {
			t.Errorf("expected %v, got %v", e, err)
		}
	}()
	_ = pio.Catch(func() error {
		panic(e)
	})
}
