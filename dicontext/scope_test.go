package dicontext_test

import (
	"context"
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/dicontext"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/sectrean/di-kit/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Scope(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := dicontext.WithScope(context.Background(), c)
		scope := dicontext.Scope(ctx)

		assert.Same(t, c, scope)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		scope := dicontext.Scope(ctx)
		assert.Nil(t, scope)
	})
}

func Test_Resolve(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := dicontext.WithScope(context.Background(), c)
		got, err := dicontext.Resolve[testtypes.InterfaceA](ctx)

		assert.Equal(t, &testtypes.StructA{}, got)
		assert.NoError(t, err)
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

		ctx := dicontext.WithScope(context.Background(), c)
		got, err := dicontext.Resolve[testtypes.InterfaceA](ctx, di.WithTag("tag"))

		assert.Equal(t, &testtypes.StructA{}, got)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := dicontext.WithScope(context.Background(), c)
		got, err := dicontext.Resolve[testtypes.InterfaceA](ctx)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err,
			"dicontext.Resolve: di.Container.Resolve testtypes.InterfaceA: service not registered")
	})

	t.Run("scope not found", func(t *testing.T) {
		ctx := context.Background()
		got, err := dicontext.Resolve[testtypes.InterfaceA](ctx)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err,
			"dicontext.Resolve testtypes.InterfaceA: scope not found on context")
	})
}

func Test_MustResolve(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := dicontext.WithScope(context.Background(), c)
		got := dicontext.MustResolve[testtypes.InterfaceA](ctx)

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

		ctx := dicontext.WithScope(context.Background(), c)
		got := dicontext.MustResolve[testtypes.InterfaceA](ctx, di.WithTag("tag"))

		assert.Equal(t, &testtypes.StructA{}, got)
	})

	t.Run("scope not found", func(t *testing.T) {
		ctx := context.Background()

		assert.PanicsWithError(t, "dicontext.Resolve testtypes.InterfaceA: scope not found on context", func() {
			_ = dicontext.MustResolve[testtypes.InterfaceA](ctx)
		})
	})

	t.Run("error", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := dicontext.WithScope(context.Background(), c)

		assert.PanicsWithError(t, "dicontext.Resolve: di.Container.Resolve testtypes.InterfaceA: service not registered", func() {
			_ = dicontext.MustResolve[testtypes.InterfaceA](ctx)
		})
	})
}
