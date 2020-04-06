// Package testutils contains convenient testing checkers that compare a produced
// value against an expected value (or condition).
// There are value checks like `CheckEqual(expected, produced, t)``, and
// checks that should run deferred like `defer ShouldPanic(t)`.
//
package testutils

import (
	"reflect"
	"testing"
)

// CheckEqual checks if two values are deeply equal and calls t.Fatalf if not
func CheckEqual(expected interface{}, got interface{}, t *testing.T) {
	t.Helper()
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("Expected: %v, got %v", expected, got)
	}
}

// CheckNil checks if value is nil
func CheckNil(got interface{}, t *testing.T) {
	t.Helper()
	if !reflect.ValueOf(got).IsNil() {
		t.Fatalf("Expected: nil, got %v", got)
	}
}

// CheckNotNil checks if value is not nil
func CheckNotNil(got interface{}, t *testing.T) {
	t.Helper()
	if reflect.ValueOf(got).IsNil() {
		t.Fatalf("Expected: not nil, got nil")
	}
}

// CheckError checks if there is an error
func CheckError(got error, t *testing.T) {
	t.Helper()
	if got == nil {
		t.Fatalf("Expected: error, got %v", got)
	}
}

// CheckNotError checks if error value is not nil
func CheckNotError(got error, t *testing.T) {
	t.Helper()
	if got != nil {
		t.Fatalf("Expected: no error, got %v", got)
	}
}

// CheckTrue checks if value is true
func CheckTrue(got bool, t *testing.T) {
	t.Helper()
	if !got {
		t.Fatalf("Expected: true, got %v", got)
	}
}

// CheckFalse checks if value is false
func CheckFalse(got bool, t *testing.T) {
	t.Helper()
	if got {
		t.Fatalf("Expected: false, got %v", got)
	}
}
