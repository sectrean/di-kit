package di_test

import (
	"context"

	"github.com/sectrean/di-kit"
)

func NewScopeFactory[T any](scope di.Scope, fn scopeFactoryFunc[T]) *ScopeFactory[T] {
	return &ScopeFactory[T]{
		scope: scope,
		fn:    fn,
	}
}

type scopeFactoryFunc[T any] func(context.Context, di.Scope) (T, error)

type ScopeFactory[T any] struct {
	scope di.Scope
	fn    scopeFactoryFunc[T]
}

func (f *ScopeFactory[T]) Build(ctx context.Context) (T, error) {
	return f.fn(ctx, f.scope)
}
