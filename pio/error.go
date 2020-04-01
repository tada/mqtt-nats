package pio

// An Error is used as the argument to a panic by functions in this package that encounter an error
type Error struct {
	// Cause is the original error
	Cause error
}

// Error returns the result of calling Error() on the contained Cause
func (e *Error) Error() string {
	return e.Cause.Error()
}
