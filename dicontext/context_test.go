package dicontext_test

import (
	"context"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/dicontext"
	"github.com/johnrutherford/di-kit/internal/testtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Scope(t *testing.T) {
	t.Run("with scope", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := dicontext.WithScope(context.Background(), c)
		scope := dicontext.Scope(ctx)

		assert.Same(t, c, scope)
	})

	t.Run("no scope", func(t *testing.T) {
		ctx := context.Background()
		scope := dicontext.Scope(ctx)
		assert.Nil(t, scope)
	})
}

func Test_Resolve(t *testing.T) {
	t.Run("resolve", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := dicontext.WithScope(context.Background(), c)
		got, err := dicontext.Resolve[testtypes.InterfaceA](ctx)

		assert.Equal(t, &testtypes.StructA{}, got)
		assert.NoError(t, err)
	})

	t.Run("resolve with tag", func(t *testing.T) {
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

	t.Run("resolve error", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := dicontext.WithScope(context.Background(), c)
		got, err := dicontext.Resolve[testtypes.InterfaceA](ctx)
		// TODO: Log error messages

		assert.Nil(t, got)
		assert.EqualError(t, err,
			"dicontext.Resolve: di.Container.Resolve testtypes.InterfaceA: service not registered")
	})

	t.Run("no scope", func(t *testing.T) {
		ctx := context.Background()
		got, err := dicontext.Resolve[testtypes.InterfaceA](ctx)

		assert.Nil(t, got)
		assert.EqualError(t, err,
			"dicontext.Resolve testtypes.InterfaceA: scope not found on context")
	})
}

func Test_MustResolve(t *testing.T) {
	t.Run("resolve", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := dicontext.WithScope(context.Background(), c)
		got := dicontext.MustResolve[testtypes.InterfaceA](ctx)

		assert.Equal(t, &testtypes.StructA{}, got)
	})

	t.Run("no scope", func(t *testing.T) {
		ctx := context.Background()

		assert.PanicsWithError(t, "dicontext.Resolve testtypes.InterfaceA: scope not found on context", func() {
			_ = dicontext.MustResolve[testtypes.InterfaceA](ctx)
		})
	})

	t.Run("resolve error", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := dicontext.WithScope(context.Background(), c)

		assert.PanicsWithError(t, "dicontext.Resolve: di.Container.Resolve testtypes.InterfaceA: service not registered", func() {
			_ = dicontext.MustResolve[testtypes.InterfaceA](ctx)
		})
	})
}
