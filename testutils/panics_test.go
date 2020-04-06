package testutils

import (
	"errors"
	"testing"
)

func TestShouldNotPanic(t *testing.T) {
	ensureFailed(t, func(ft *testing.T) {
		defer ShouldNotPanic(ft)
		panic(errors.New("but it did"))
	})
}

func TestShouldPanic(t *testing.T) {
	ensureFailed(t, func(ft *testing.T) {
		defer ShouldPanic(ft)
		// but didn't
	})
}
