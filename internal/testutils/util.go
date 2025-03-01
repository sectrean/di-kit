package testutils

import (
	"context"
	"sync"
	"testing"
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

// RunParallel runs a function in parallel with the given concurrency.
func RunParallel(concurrency int, f func(int)) {
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

// CollectChannel collects all values from a channel and returns them in a slice.
func CollectChannel[V any](ch <-chan V) []V {
	//nolint:prealloc // No way of knowing the number of values in the channel
	var values []V
	for v := range ch {
		values = append(values, v)
	}

	return values
}
