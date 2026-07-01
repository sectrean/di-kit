// Package ditest provides testing utilities for the di-kit package.
package ditest

import (
	"reflect"
	"testing"

	"github.com/sectrean/di-kit"
)

// TestingT is an interface that defines the methods required for testing in Go.
type TestingT interface {
	// Helper marks the calling function as a test helper function.
	Helper()
	// Errorf formats its arguments according to the format, analogous to fmt.Errorf, and records the error in the test.
	Errorf(format string, args ...any)
}

var _ TestingT = (*testing.T)(nil)

// AssertContains asserts that the given scope contains the specified type T.
func AssertContains[Service any](t TestingT, s di.Scope) bool {
	t.Helper()

	typ := reflect.TypeFor[Service]()
	if s.Contains(typ) {
		return true
	}

	t.Errorf("ditest.AssertContains: Scope should contain type %s", typ)
	return false
}

// AssertNotContains asserts that the given scope does not contain the specified type T.
func AssertNotContains[Service any](t TestingT, s di.Scope) bool {
	t.Helper()

	typ := reflect.TypeFor[Service]()
	if !s.Contains(typ) {
		return true
	}

	t.Errorf("ditest.AssertNotContains: Scope should not contain type %s", typ)
	return false
}
