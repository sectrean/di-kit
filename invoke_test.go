package di_test

import (
	"context"
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/internal/errors"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/sectrean/di-kit/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Invoke(t *testing.T) {
	t.Run("not func", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		err = di.Invoke(ctx, c, 1234)
		testutils.LogError(t, err)

		assert.EqualError(t, err, "di.Invoke int: fn must be a function")
	})

	t.Run("dependency nil", func(t *testing.T) {
		calls := 0

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return nil }),
		)
		require.NoError(t, err)

		ctx := context.Background()
		err = di.Invoke(ctx, c, func(a testtypes.InterfaceA) {
			assert.Nil(t, a)
			calls++
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("one param", func(t *testing.T) {
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
		testutils.LogError(t, err)

		assert.EqualError(t, err, "test invoke error")
	})

	t.Run("return nil error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		err = di.Invoke(ctx, c, func(testtypes.InterfaceA) error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("resolve error", func(t *testing.T) {
		c, err := di.NewContainer()

		require.NoError(t, err)

		ctx := context.Background()
		err = di.Invoke(ctx, c, func(testtypes.InterfaceA) {})
		testutils.LogError(t, err)

		assert.EqualError(t, err, "di.Invoke func(testtypes.InterfaceA): di.Container.Resolve testtypes.InterfaceA: service not registered")
	})

	t.Run("with context", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := testutils.ContextWithTestValue(context.Background(), "value")
		err = di.Invoke(ctx, c, func(ctx2 context.Context, a testtypes.InterfaceA) {
			assert.Same(t, ctx, ctx2)
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

		err = di.Invoke(ctx, c, func() {})
		testutils.LogError(t, err)

		assert.EqualError(t, err, "di.Invoke func(): context canceled")
	})

	t.Run("with context error during resolve", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)

		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = di.Invoke(ctx, c, func(context.Context, testtypes.InterfaceA) {})
		testutils.LogError(t, err)

		assert.EqualError(t, err, "di.Invoke func(context.Context, testtypes.InterfaceA): di.Container.Resolve testtypes.InterfaceA: context canceled")
	})

	t.Run("with tagged", func(t *testing.T) {
		a := &testtypes.StructA{}

		c, err := di.NewContainer(
			di.WithService(a,
				di.As[testtypes.InterfaceA](),
				di.WithTag("tag"),
			),
			di.WithService(testtypes.NewInterfaceA),
		)

		require.NoError(t, err)

		ctx := context.Background()
		a2, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithTag("tag"))
		assert.Same(t, a, a2)
		assert.NoError(t, err)

		err = di.Invoke(ctx, c,
			func(aa testtypes.InterfaceA) {
				assert.Same(t, a, aa)
			},
			di.WithTagged[testtypes.InterfaceA]("tag"),
		)
		assert.NoError(t, err)
	})

	t.Run("with tagged dep not found", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		err = di.Invoke(ctx, c,
			func(testtypes.InterfaceA) {},
			di.WithTagged[testtypes.InterfaceB]("tag"),
		)
		testutils.LogError(t, err)

		assert.EqualError(t, err, "di.Invoke func(testtypes.InterfaceA): WithTagged testtypes.InterfaceB: parameter not found")
	})
}
