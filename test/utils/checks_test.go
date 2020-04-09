package utils

import (
	"io"
	"testing"
)

func TestCheckEqual(t *testing.T) {
	EnsureFailed(t, func(ft *testing.T) {
		CheckEqual("a", "b", ft)
	})
}

func TestCheckNil(t *testing.T) {
	EnsureFailed(t, func(ft *testing.T) {
		CheckNil([]byte{0}, ft)
	})
}

func TestCheckNotNil(t *testing.T) {
	EnsureFailed(t, func(ft *testing.T) {
		CheckNotNil(nil, ft)
	})
}

func TestCheckError(t *testing.T) {
	EnsureFailed(t, func(ft *testing.T) {
		CheckError(nil, ft)
	})
}

func TestCheckNotError(t *testing.T) {
	EnsureFailed(t, func(ft *testing.T) {
		CheckNotError(io.ErrUnexpectedEOF, ft)
	})
}

func TestCheckTrue(t *testing.T) {
	EnsureFailed(t, func(ft *testing.T) {
		CheckTrue(false, ft)
	})
}

func TestCheckFalse(t *testing.T) {
	EnsureFailed(t, func(ft *testing.T) {
		CheckFalse(true, ft)
	})
}
