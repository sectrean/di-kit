package di_test

import (
	"context"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/errors"
	"github.com/johnrutherford/di-kit/internal/testtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Invoke(t *testing.T) {
	t.Parallel()

	t.Run("not func", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		err = di.Invoke(ctx, c, 1234)
		LogError(t, err)

		assert.EqualError(t, err, "invoke int: fn must be a function")
	})

	t.Run("one arg", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		called := false

		err = di.Invoke(ctx, c, func(a testtypes.InterfaceA) {
			assert.NotNil(t, a)
			called = true
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("return error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		err = di.Invoke(ctx, c, func(testtypes.InterfaceA) error {
			return errors.New("test invoke error")
		})
		LogError(t, err)

		assert.EqualError(t, err, "test invoke error")
	})

	t.Run("resolve error", func(t *testing.T) {
		c, err := di.NewContainer()

		require.NoError(t, err)

		ctx := context.Background()
		err = di.Invoke(ctx, c, func(testtypes.InterfaceA) {})
		LogError(t, err)

		assert.EqualError(t, err, "invoke func(testtypes.InterfaceA): resolve testtypes.InterfaceA: type not registered")
	})

	t.Run("with context", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.WithValue(context.Background(), "key", "value")
		err = di.Invoke(ctx, c, func(ctx2 context.Context, a testtypes.InterfaceA) {
			assert.Equal(t, ctx, ctx2)
			assert.NotNil(t, a)
		})
		assert.NoError(t, err)
	})

	t.Run("with context error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)

		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = di.Invoke(ctx, c, func(context.Context, testtypes.InterfaceA) {})
		LogError(t, err)

		assert.EqualError(t, err, "invoke func(context.Context, testtypes.InterfaceA): context canceled")
	})

	t.Run("with keyed", func(t *testing.T) {
		a := &testtypes.StructA{}

		c, err := di.NewContainer(
			di.WithService(a,
				di.As[testtypes.InterfaceA](),
				di.WithKey("key"),
			),
			di.WithService(testtypes.NewInterfaceA),
		)

		require.NoError(t, err)

		ctx := context.Background()
		a2, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithKey("key"))
		assert.Same(t, a, a2)
		assert.NoError(t, err)

		err = di.Invoke(ctx, c,
			func(aa testtypes.InterfaceA) {
				assert.Same(t, a, aa)
			},
			di.WithKeyed[testtypes.InterfaceA]("key"),
		)
		assert.NoError(t, err)
	})

	t.Run("with keyed dep not found", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		err = di.Invoke(ctx, c,
			func(testtypes.InterfaceA) {},
			di.WithKeyed[testtypes.InterfaceB]("key"),
		)
		LogError(t, err)

		assert.EqualError(t, err, "invoke func(testtypes.InterfaceA): with keyed testtypes.InterfaceB: argument not found")
	})
}
