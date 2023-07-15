package di

import "context"

type Closer interface {
	Close(ctx context.Context) error
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

// getCloser returns a Closer if val implements one of the following Close method signatures:
//
// - Close(context.Context) error
//
// - Close(context.Context)
//
// - Close() error
//
// - Close()
//
// Otherwise, returns nil.
func getCloser(val any) (Closer, bool) {
	if val == nil {
		return nil, false
	}

	if c, ok := val.(Closer); ok {
		return c, true
	} else if c, ok := val.(closerWithContextNoError); ok {
		return closerWithContextNoErrorWrapper{c}, true
	} else if c, ok := val.(closerNoContextWithError); ok {
		return closerNoContextWithErrorWrapper{c}, true
	} else if c, ok := val.(closerNoContextNoError); ok {
		return closerNoContextNoErrorWrapper{c}, true
	}

	return nil, false
}

type closerNoContextNoErrorWrapper struct {
	c closerNoContextNoError
}

func (w closerNoContextNoErrorWrapper) Close(_ context.Context) error {
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

func (w closerNoContextWithErrorWrapper) Close(_ context.Context) error {
	return w.c.Close()
}
