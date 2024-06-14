package di

import (
	"context"
	"reflect"
	"testing"

	"github.com/johnrutherford/di-kit/internal/testtypes"
)

var (
	InterfaceAType = reflect.TypeFor[testtypes.InterfaceA]()
	InterfaceBType = reflect.TypeFor[testtypes.InterfaceB]()

	InterfaceAKey = serviceKey{Type: InterfaceAType}
	InterfaceBKey = serviceKey{Type: InterfaceBType}
)

// LogError is a test helper function to log an error message if it is not nil.
//
// This is to help make sure our error messages are helpful and informative.
func LogError(t *testing.T, err error) {
	if err == nil {
		return
	}

	t.Helper()
	t.Logf("error message:\n%v", err)
}

func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func ContextCanceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}

func ContextDeadlineExceeded() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), -1)
	cancel()

	return ctx
}
