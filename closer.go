package di

import (
	"context"
)

// Closer is used to close a resolved service. If a service implements Closer, the Close function
// will be called when the container is closed.
//
// Any of these Close method signatures are also supported:
//
// - Close(context.Context) error
//
// - Close(context.Context)
//
// - Close() error
//
// - Close()
type Closer interface {
	Close(ctx context.Context) error
}

// CloseFunc is a function that implements the Closer interface.
type CloseFunc func(context.Context) error

// Close calls the function.
func (f CloseFunc) Close(ctx context.Context) error {
	return f(ctx)
}

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
