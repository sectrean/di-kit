package di

import (
	"context"
	"reflect"

	"github.com/johnrutherford/di-kit/internal/testtypes"
)

type futureService[T any] struct {
	t   reflect.Type
	tag any
	svc service
}

func newFutureService[T any](svc service) *futureService[T] {
	return &futureService[T]{
		t:   reflect.TypeFor[Future[T]](),
		svc: svc,
	}
}

var lazyDeps = []serviceKey{
	{Type: contextType},
	{Type: scopeType},
}

func (*futureService[T]) AddAlias(alias reflect.Type) error {
	panic("not supported")
}

func (*futureService[T]) Aliases() []reflect.Type {
	return nil
}

func (*futureService[T]) Dependencies() []serviceKey {
	return lazyDeps
}

func (*futureService[T]) GetCloser(val any) Closer {
	return nil
}

func (*futureService[T]) GetValue(deps []reflect.Value) (any, error) {
	ctx := deps[0].Interface().(context.Context)
	scope := deps[1].Interface().(Scope)

	return newLazyFuture[T](ctx, scope), nil
}

func (*futureService[T]) Lifetime() Lifetime {
	return Scoped
}

func (s *futureService[T]) Tag() any {
	return s.tag
}

func (s *futureService[T]) Type() reflect.Type {
	return s.t
}

func (*futureService[T]) setCloserFactory(closerFactory) {
	panic("not supported")
}

// setLifetime implements service.
func (*futureService[T]) setLifetime(Lifetime) {
	panic("not supported")
}

// setTag implements service.
func (s *futureService[T]) setTag(tag any) {
	s.tag = tag
}

var _ service = (*futureService[testtypes.InterfaceA])(nil)
