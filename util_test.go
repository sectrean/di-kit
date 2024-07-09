package di_test

import (
	"context"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/testtypes"
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

type ctxKey struct{}

// ContextWithTestValue returns a context with the provided value
func ContextWithTestValue(ctx context.Context, val any) context.Context {
	return context.WithValue(ctx, ctxKey{}, val)
}

func NewTestFactory(scope di.Scope, fn factoryFunc) *TestFactory {
	return &TestFactory{
		scope: scope,
		fn:    fn,
	}
}

type factoryFunc func(context.Context, di.Scope) testtypes.InterfaceA

type TestFactory struct {
	scope di.Scope
	fn    factoryFunc
}

func (f *TestFactory) Build(ctx context.Context) testtypes.InterfaceA {
	return f.fn(ctx, f.scope)
}
