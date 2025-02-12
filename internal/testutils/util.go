package testutils

import (
	"sync"
	"testing"
)

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
