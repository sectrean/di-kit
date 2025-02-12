package testtypes

import (
	"context"

	"github.com/sectrean/di-kit"
)

func NewTestFactory[T any](scope di.Scope, fn factoryFunc[T]) *TestFactory[T] {
	return &TestFactory[T]{
		scope: scope,
		fn:    fn,
	}
}

type factoryFunc[T any] func(context.Context, di.Scope) (T, error)

type TestFactory[T any] struct {
	scope di.Scope
	fn    factoryFunc[T]
}

func (f *TestFactory[T]) Build(ctx context.Context) (T, error) {
	return f.fn(ctx, f.scope)
}
