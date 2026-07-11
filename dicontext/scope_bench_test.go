//go:build go1.24

package dicontext_test

import (
	"context"
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/dicontext"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/stretchr/testify/require"
)

func Benchmark_Resolve(b *testing.B) {
	b.Run("singleton", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)

		// Store the scope on the context once, as a request-scoped handler would.
		ctx := dicontext.WithScope(context.Background(), c)

		got, err := dicontext.Resolve[testtypes.InterfaceA](ctx)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		for b.Loop() {
			_, _ = dicontext.Resolve[testtypes.InterfaceA](ctx)
		}
	})
}
