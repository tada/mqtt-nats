package pio

import (
	"errors"
	"testing"
)

func TestError(t *testing.T) {
	err := errors.New("the error")
	ex := Error{err}
	if err.Error() != ex.Error() {
		t.Error("Error produces wrong error string")
	}
}
