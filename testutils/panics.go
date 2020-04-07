// +build citest

package testutils

import "testing"

// ShouldNotPanic is used to assert that a function does not panic
// Usage: defer testutils.ShouldNotPanic(t) at the point where the rest is expected to not panic
func ShouldNotPanic(t *testing.T) {
	t.Helper()
	if r := recover(); r != nil {
		t.Error("Unexpected panic")
	}
}

// ShouldPanic is used to assert that a function does panic
// Usage: defer testutils.ShouldNotPanic(t) at the point where the rest is expected to panic
func ShouldPanic(t *testing.T) {
	t.Helper()
	if r := recover(); r == nil {
		t.Error("Expected panic but got none")
	}
}
