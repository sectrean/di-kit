package di_test

import (
	"context"
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/internal/testtypes"
	. "github.com/sectrean/di-kit/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ValidateDependencies(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		assert.NoError(t, err)
	})

	t.Run("invalid service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceB),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		LogError(t, err)

		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func(testtypes.InterfaceA) testtypes.InterfaceB: "+
			"dependency testtypes.InterfaceA: service not registered",
		)
	})

	t.Run("scoped service not validated on parent", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceC, di.Scoped),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		assert.NoError(t, err)
	})

	t.Run("dependency cycle", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func(context.Context, testtypes.InterfaceC) testtypes.InterfaceB { return nil }),
			di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceC { return nil }),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		LogError(t, err)

		// The exact error message is non-deterministic because it depends on map iteration order
		assert.ErrorContains(t, err, "dependency cycle detected")
	})

	t.Run("dependency cycle single type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(context.Context, testtypes.InterfaceA) testtypes.InterfaceA { return nil }),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		LogError(t, err)

		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func(context.Context, testtypes.InterfaceA) testtypes.InterfaceA: "+
			"dependency testtypes.InterfaceA: dependency cycle detected",
		)
	})

	t.Run("slice dependency", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func([]testtypes.InterfaceA) testtypes.InterfaceB {
				return testtypes.StructB{}
			}),
			di.WithService(func([]testtypes.InterfaceC) testtypes.InterfaceD {
				return testtypes.StructD{}
			}),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		LogError(t, err)

		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func([]testtypes.InterfaceC) testtypes.InterfaceD: "+
			"dependency testtypes.InterfaceC: service not registered",
		)
	})

	t.Run("variadic dependency optional", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func(...testtypes.InterfaceA) testtypes.InterfaceB {
				return testtypes.StructB{}
			}),
			di.WithService(func(...testtypes.InterfaceC) testtypes.InterfaceD {
				return testtypes.StructD{}
			}),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		assert.NoError(t, err)
	})

	t.Run("child scope", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.Scoped),
		)
		require.NoError(t, err)

		scope, err := c.NewScope()
		require.NoError(t, err)

		err = di.ValidateContainer(scope)
		assert.NoError(t, err)
	})

	t.Run("child scope dependency not registered", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceB, di.Scoped),
		)
		require.NoError(t, err)

		scope, err := c.NewScope()
		require.NoError(t, err)

		err = di.ValidateContainer(scope)
		LogError(t, err)

		assert.NotNil(t, scope)
		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func(testtypes.InterfaceA) testtypes.InterfaceB: "+
			"dependency testtypes.InterfaceA: service not registered",
		)
	})

	t.Run("child scope dependency cycle", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceC { return nil }, di.Scoped),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(func(testtypes.InterfaceC) testtypes.InterfaceB { return nil }),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(scope)
		LogError(t, err)

		// The exact error message is non-deterministic because it depends on map iteration order
		assert.ErrorContains(t, err, "dependency cycle detected")
	})
}
