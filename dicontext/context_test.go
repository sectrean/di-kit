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

	t.Run("resolve with key", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithKey("key")),
			di.WithService(func() testtypes.InterfaceA {
				panic("should not be called")
			}),
		)
		require.NoError(t, err)

		ctx := dicontext.WithScope(context.Background(), c)

		got, err := dicontext.Resolve[testtypes.InterfaceA](ctx, di.WithKey("key"))
		assert.Equal(t, &testtypes.StructA{}, got)
		assert.NoError(t, err)
	})

	t.Run("resolve error", func(t *testing.T) {
		ctx := context.Background()

		got, err := dicontext.Resolve[testtypes.InterfaceA](ctx)
		assert.Nil(t, got)
		assert.EqualError(t, err,
			"resolve testtypes.InterfaceA from context: scope not found on context")
	})

	t.Run("no scope", func(t *testing.T) {
		ctx := context.Background()

		got, err := dicontext.Resolve[testtypes.InterfaceA](ctx)
		assert.Nil(t, got)
		assert.EqualError(t, err,
			"resolve testtypes.InterfaceA from context: scope not found on context")
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

		assert.PanicsWithError(t, "resolve testtypes.InterfaceA from context: scope not found on context", func() {
			_ = dicontext.MustResolve[testtypes.InterfaceA](ctx)
		})
	})

	t.Run("resolve error", func(t *testing.T) {
		ctx := context.Background()

		assert.PanicsWithError(t, "resolve testtypes.InterfaceA from context: scope not found on context", func() {
			_ = dicontext.MustResolve[testtypes.InterfaceA](ctx)
		})
	})
}
