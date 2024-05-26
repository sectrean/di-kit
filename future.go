package di

import (
	"context"
	"sync"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// Future represents a value that has not been resolved from the Scope yet.
// The value is resolved when [Result] is called.
type Future[T any] interface {
	// Result returns the resolved value or an error if the value could not be resolved.
	Result() (T, error)
}

type lazyFuture[T any] struct {
	fn func() (T, error)
}

func newLazyFuture[T any](ctx context.Context, scope Scope) lazyFuture[T] {
	return lazyFuture[T]{
		fn: sync.OnceValues(func() (T, error) {
			return Resolve[T](ctx, scope)
		}),
	}
}

func (f lazyFuture[T]) Result() (T, error) {
	val, err := f.fn()
	return val, errors.Wrap(err, "lazy future result")
}

var _ Future[any] = (*lazyFuture[any])(nil)

type resolveFuture struct {
	val  any
	err  error
	done chan struct{}
}

func newFuture() *resolveFuture {
	return &resolveFuture{
		done: make(chan struct{}),
	}
}

func (f *resolveFuture) setResult(val any, err error) {
	f.val = val
	f.err = err
	close(f.done)
}

func (f *resolveFuture) Result() (any, error) {
	<-f.done
	return f.val, f.err
}
