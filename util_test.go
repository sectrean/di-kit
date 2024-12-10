package di_test

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/testtypes"
)

var (
	TypeStructA    = reflect.TypeFor[testtypes.StructA]()
	TypeStructAPtr = reflect.TypeFor[*testtypes.StructA]()
	TypeInterfaceA = reflect.TypeFor[testtypes.InterfaceA]()

	TypeStructB    = reflect.TypeFor[testtypes.StructB]()
	TypeStructBPtr = reflect.TypeFor[*testtypes.StructB]()
	TypeInterfaceB = reflect.TypeFor[testtypes.InterfaceB]()

	TypeStructC    = reflect.TypeFor[testtypes.StructC]()
	TypeStructCPtr = reflect.TypeFor[*testtypes.StructC]()
	TypeInterfaceC = reflect.TypeFor[testtypes.InterfaceC]()

	TypeStructD    = reflect.TypeFor[testtypes.StructD]()
	TypeStructDPtr = reflect.TypeFor[*testtypes.StructD]()
	TypeInterfaceD = reflect.TypeFor[testtypes.InterfaceD]()
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

// ContextWithTestValue returns a context with the provided value.
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

func runConcurrent(concurrency int, f func(int)) {
	wg := sync.WaitGroup{}
	wg.Add(concurrency)

	for i := range concurrency {
		go func() {
			defer wg.Done()
			f(i)
		}()
	}

	wg.Wait()
}
