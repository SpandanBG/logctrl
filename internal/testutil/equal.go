// Package testutil holds assertion helpers shared by every package's tests.
package testutil

import "testing"

// Equal - fails the test when `expected` and `actual` differ.
func Equal[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected == actual {
		return
	}
	t.Errorf(
		"expected equal\n\texpected:\t%v\n\tactual:\t%v",
		expected, actual,
	)
}
