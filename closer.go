package di

import (
	"context"
	"reflect"

	"github.com/sectrean/di-kit/internal/errors"
)

// Closer is an interface for cleaning up resources associated with a service when the
// [Container] is closed.
//
// A [Container] will automatically close services that implement any of these
// Close function signatures:
//
//	Close(context.Context) error
//	Close(context.Context)
//	Close() error
//	Close()
//
// This is the default behavior for function services.
// When the [Container] creates a service, it will be responsible for closing it.
// Use the [IgnoreClose] option to ignore a Close method for a service.
//
// Value services are not closed by default.
// Since value services are not created by the [Container], it is assumed that
// their lifetime will be managed outside of the [Container].
// Use the [WithClose] option to automatically close a value service when the [Container] is closed.
//
// Use the [WithCloseFunc] option to specify a custom function to close a service.
type Closer interface {
	// Close resources owned by the service.
	Close(ctx context.Context) error
}

// WithClose is used to close a value service when the [Container] is closed.
//
// If a function service implements [Closer], or a compatible Close function signature,
// it will be called when the [Container] is closed.
//
// Value services are not closed by default.
// Use this option if you want the [Container] to call Close on a value service.
//
// See Closer for more information.
func WithClose() ServiceOption {
	return serviceOption(func(sc serviceConfig) error {
		sc.SetCloserFactory(getCloser)
		return nil
	})
}

// IgnoreClose will not close the service when the [Container] is closed.
// This is useful when you want to manage the lifecycle of a service outside of the [Container].
//
// Function services are closed by default.
// Use this option if you do not want a function service to be closed by the [Container].
// This is the default behavior for value services.
//
// See [Closer] for more information.
func IgnoreClose() ServiceOption {
	return serviceOption(func(sc serviceConfig) error {
		sc.SetCloserFactory(nil)
		return nil
	})
}

type closerFactory func(val any) Closer

// WithCloseFunc configures a custom function to call to close the service when the [Container] is closed.
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
// See [Closer] for more information.
//
// This option will return an error if the service type is not assignable to Service.
func WithCloseFunc[Service any](f func(context.Context, Service) error) ServiceOption {
	return serviceOption(func(sc serviceConfig) error {
		if !sc.Type().AssignableTo(reflect.TypeFor[Service]()) {
			return errors.Errorf("WithCloseFunc: service type %s is not assignable to %s",
				sc.Type(), reflect.TypeFor[Service]())
		}

		sc.SetCloserFactory(func(val any) Closer {
			return closeFunc(func(ctx context.Context) error {
				return f(ctx, val.(Service))
			})
		})
		return nil
	})
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
