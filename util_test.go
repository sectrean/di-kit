package di_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/testtypes"
)

var (
	InterfaceAType = reflect.TypeFor[testtypes.InterfaceA]()
	InterfaceBType = reflect.TypeFor[testtypes.InterfaceB]()
	InterfaceCType = reflect.TypeFor[testtypes.InterfaceC]()
	InterfaceDType = reflect.TypeFor[testtypes.InterfaceD]()
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

type Factory struct {
	scope di.Scope
}

func (f *Factory) BuildA(ctx context.Context, _ string) testtypes.InterfaceA {
	a, err := di.Resolve[testtypes.InterfaceA](ctx, f.scope)
	if err != nil {
		panic(err)
	}

	return a
}
