// Package pio contains io related functions that instead of returning an error, panics with
// an Error which has a Cause (the original error). The Catch function can then be used to
// recover the panic and return the original error
package pio
