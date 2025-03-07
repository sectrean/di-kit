package di_test

import (
	"context"
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_MustResolve(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got := di.MustResolve[testtypes.InterfaceA](ctx, c)
		assert.Equal(t, &testtypes.StructA{}, got)
	})

	t.Run("WithTag", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithTag("tag")),
			di.WithService(func() testtypes.InterfaceA {
				assert.Fail(t, "should not be called")
				return nil
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got := di.MustResolve[testtypes.InterfaceA](ctx, c, di.WithTag("tag"))
		assert.Equal(t, &testtypes.StructA{}, got)
	})

	t.Run("error", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		assert.PanicsWithError(t,
			"di.Container.Resolve testtypes.InterfaceA: service not registered",
			func() {
				di.MustResolve[testtypes.InterfaceA](ctx, c)
			},
		)
	})
}
