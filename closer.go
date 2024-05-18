package di

import (
	"context"
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// Closer is used to close a service when closing the Container.
//
// If a resolved service implements Closer, or one of the other compatible function signatures,
// the Close function will be called when the Container is closed.
//
// Any of these Close method signatures are supported:
//
//	Close(context.Context) error
//	Close(context.Context)
//	Close() error
//	Close()
//
// See related options:
//   - [IgnoreCloser]
//   - [WithCloser]
//   - [WithCloseFunc]
type Closer interface {
	Close(ctx context.Context) error
}

// WithCloser is used to close a service when the Container is closed.
//
// If a service implements Closer, or one of the other compatible Close function signatures, the Close
// function will be called when the Container is closed.
//
// Value services are not closed by default. To close a value service, use this option.
func WithCloser() RegisterValueOption {
	return closerOption(func(reflect.Type) (closerFactory, error) {
		return getCloser, nil
	})
}

// TODO: Should we implement support for Shutdown(context.Context) error?

// IgnoreCloser is used when you do not want a service that implements Closer, or another
// supported Close function signature, to be closed when the Container is closed.
//
// This is useful when you want to manage the lifecycle of a service outside of the Container.
func IgnoreCloser() RegisterFuncOption {
	return closerOption(func(reflect.Type) (closerFactory, error) {
		return nil, nil
	})
}

// CloserOption is used to configure the behavior for closing a resolved service.
// This can be used with [RegisterFunc] and [RegisterValue].
//
// Available options:
//   - [WithCloseFunc]
//
// See also:
//   - [Closer]
//   - [WithCloser]
//   - [IgnoreCloser]
type CloserOption interface {
	RegisterFuncOption
	RegisterValueOption
}

type closerFactory func(val any) Closer

type closerOption func(t reflect.Type) (closerFactory, error)

func (o closerOption) applyFuncService(s *funcService) error {
	closerFactory, err := o(s.t)
	if err != nil {
		return err
	}
	s.closerFactory = closerFactory

	return nil
}

func (o closerOption) applyValueService(s *valueService) error {
	closerFactory, err := o(s.t)
	if err != nil {
		return err
	}
	s.closerFactory = closerFactory

	return nil
}

var _ CloserOption = (closerOption)(nil)

// WithCloseFunc can be used to set a custom function to call for a Service when the Container is closed.
//
// This is useful if a service has a method called Shutdown or Stop instead of Close that should be
// used to close the service.
//
// Example:
//
//	di.WithCloseFunc(func(ctx context.Context, s *http.Server) error {
//		return s.Shutdown(ctx)
//	})
//
// This can also be used to close a service registered with a value rather than a function.
// Services registered with a value will not be closed by default.
//
// This option will return an error if the service type is not assignable to T.
func WithCloseFunc[T any](f func(context.Context, T) error) CloserOption {
	return closeFuncOption[T]{f}
}

type closeFuncOption[T any] struct {
	f func(context.Context, T) error
}

func (o closeFuncOption[T]) applyFuncService(s *funcService) error {
	if !s.t.AssignableTo(reflect.TypeFor[T]()) {
		return errors.Errorf("service type %s is not assignable to close func type %s",
			s.t, reflect.TypeFor[T]())
	}

	s.closerFactory = func(val any) Closer {
		return closeFunc(func(ctx context.Context) error {
			return o.f(ctx, val.(T))
		})
	}
	return nil
}

func (o closeFuncOption[T]) applyValueService(s *valueService) error {
	t := reflect.TypeFor[T]()

	if !s.t.AssignableTo(t) {
		return errors.Errorf("service type %s is not assignable to close func type %s",
			s.t, t)
	}

	s.closerFactory = func(val any) Closer {
		return closeFunc(func(ctx context.Context) error {
			return o.f(ctx, val.(T))
		})
	}
	return nil
}

// getCloser returns the Closer interface if the given value implements it,
// or any of the compatible Close function signatures.
func getCloser(val any) Closer {
	switch c := val.(type) {
	case Closer:
		return c
	case closerWithContextNoError:
		return closerWithContextNoErrorWrapper{c}
	case closerNoContextWithError:
		return closerNoContextWithErrorWrapper{c}
	case closerNoContextNoError:
		return closerNoContextNoErrorWrapper{c}

	default:
		return nil
	}
}

type closerWithContextNoError interface {
	Close(ctx context.Context)
}

type closerNoContextWithError interface {
	Close() error
}

type closerNoContextNoError interface {
	Close()
}

type closerNoContextNoErrorWrapper struct {
	c closerNoContextNoError
}

func (w closerNoContextNoErrorWrapper) Close(context.Context) error {
	w.c.Close()
	return nil
}

type closerWithContextNoErrorWrapper struct {
	c closerWithContextNoError
}

func (w closerWithContextNoErrorWrapper) Close(ctx context.Context) error {
	w.c.Close(ctx)
	return nil
}

type closerNoContextWithErrorWrapper struct {
	c closerNoContextWithError
}

func (w closerNoContextWithErrorWrapper) Close(context.Context) error {
	return w.c.Close()
}

type closeFunc func(context.Context) error

func (f closeFunc) Close(ctx context.Context) error {
	return f(ctx)
}
