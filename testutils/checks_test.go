package testutils

import (
	"io"
	"testing"
)

func ensureFailed(t *testing.T, f func(t *testing.T)) {
	tt := testing.T{}
	x := make(chan bool, 1)
	go func() {
		defer func() { x <- true }() // GoExit runs all deferred calls
		f(&tt)
	}()
	<-x
	if !tt.Failed() {
		t.Fail()
	}
}

func TestCheckEqual(t *testing.T) {
	ensureFailed(t, func(ft *testing.T) {
		CheckEqual("a", "b", ft)
	})
}

func TestCheckNil(t *testing.T) {
	ensureFailed(t, func(ft *testing.T) {
		CheckNil([]byte{0}, ft)
	})
}

func TestCheckNotNil(t *testing.T) {
	ensureFailed(t, func(ft *testing.T) {
		CheckNotNil(nil, ft)
	})
}

func TestCheckError(t *testing.T) {
	ensureFailed(t, func(ft *testing.T) {
		CheckError(nil, ft)
	})
}

func TestCheckNotError(t *testing.T) {
	ensureFailed(t, func(ft *testing.T) {
		CheckNotError(io.ErrUnexpectedEOF, ft)
	})
}

func TestCheckTrue(t *testing.T) {
	ensureFailed(t, func(ft *testing.T) {
		CheckTrue(false, ft)
	})
}

func TestCheckFalse(t *testing.T) {
	ensureFailed(t, func(ft *testing.T) {
		CheckFalse(true, ft)
	})
}
