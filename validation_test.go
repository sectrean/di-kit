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

	t.Run("dependency not registered", func(t *testing.T) {
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

	t.Run("service with multiple invalid dependencies", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(testtypes.InterfaceD, testtypes.InterfaceC) testtypes.InterfaceB {
				return &testtypes.StructB{}
			}),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		LogError(t, err)

		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func(testtypes.InterfaceD, testtypes.InterfaceC) testtypes.InterfaceB: "+
			"dependency testtypes.InterfaceD: service not registered; dependency testtypes.InterfaceC: service not registered",
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

		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func(context.Context, testtypes.InterfaceC) testtypes.InterfaceB: "+
			"dependency testtypes.InterfaceC: service func(testtypes.InterfaceB) testtypes.InterfaceC: "+
			"dependency testtypes.InterfaceB: service func(context.Context, testtypes.InterfaceC) testtypes.InterfaceB: "+
			"dependency cycle detected\n"+
			"service func(testtypes.InterfaceB) testtypes.InterfaceC: "+
			"dependency testtypes.InterfaceB: service func(context.Context, testtypes.InterfaceC) testtypes.InterfaceB: "+
			"dependency testtypes.InterfaceC: service func(testtypes.InterfaceB) testtypes.InterfaceC: "+
			"dependency cycle detected",
		)
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
			"dependency testtypes.InterfaceA: service func(context.Context, testtypes.InterfaceA) testtypes.InterfaceA: "+
			"dependency cycle detected",
		)
	})

	t.Run("tagged service dependency cycle", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB),
			di.WithService(
				func(context.Context, testtypes.InterfaceB) testtypes.InterfaceB { return nil },
				di.WithTag("other"),
				di.WithTagged[testtypes.InterfaceB]("other"),
			),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		LogError(t, err)

		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func(context.Context, testtypes.InterfaceB) testtypes.InterfaceB {Tags [other]}: "+
			"dependency testtypes.InterfaceB {Tag other}: "+
			"service func(context.Context, testtypes.InterfaceB) testtypes.InterfaceB {Tags [other]}: "+
			"dependency cycle detected",
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
			"dependency []testtypes.InterfaceC: service not registered",
		)
	})

	t.Run("slice validates every registration in stable order", func(t *testing.T) {
		bad := func(testtypes.InterfaceD) testtypes.InterfaceA { return &testtypes.StructA{} }
		good := func() testtypes.InterfaceA { return &testtypes.StructA{} }
		dependent := func([]testtypes.InterfaceA) testtypes.InterfaceB { return &testtypes.StructB{} }

		validate := func(opts ...di.ContainerOption) string {
			c, err := di.NewContainer(opts...)
			require.NoError(t, err)

			err = di.ValidateContainer(c)
			require.Error(t, err)
			return err.Error()
		}

		badFirst := validate(
			di.WithService(bad),
			di.WithService(good),
			di.WithService(dependent),
		)
		goodFirst := validate(
			di.WithService(good),
			di.WithService(bad),
			di.WithService(dependent),
		)

		want := "di.ValidateContainer: " +
			"service func([]testtypes.InterfaceA) testtypes.InterfaceB: " +
			"dependency []testtypes.InterfaceA: " +
			"service func(testtypes.InterfaceD) testtypes.InterfaceA: " +
			"dependency testtypes.InterfaceD: service not registered\n" +
			"service func(testtypes.InterfaceD) testtypes.InterfaceA: " +
			"dependency testtypes.InterfaceD: service not registered"
		assert.Equal(t, want, badFirst)
		assert.Equal(t, want, goodFirst)
	})

	t.Run("child slice validates every inherited registration", func(t *testing.T) {
		parent, err := di.NewContainer(
			di.WithService(func(testtypes.InterfaceD) testtypes.InterfaceA {
				return &testtypes.StructA{}
			}),
			di.WithService(func() testtypes.InterfaceA { return &testtypes.StructA{} }),
		)
		require.NoError(t, err)

		child, err := parent.NewScope(
			di.WithService(func([]testtypes.InterfaceA) testtypes.InterfaceB {
				return &testtypes.StructB{}
			}),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(child)
		LogError(t, err)

		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func([]testtypes.InterfaceA) testtypes.InterfaceB: "+
			"dependency []testtypes.InterfaceA: "+
			"service func(testtypes.InterfaceD) testtypes.InterfaceA: "+
			"dependency testtypes.InterfaceD: service not registered",
		)
	})

	t.Run("variadic slice validates registered services", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(testtypes.InterfaceD) testtypes.InterfaceA {
				return &testtypes.StructA{}
			}),
			di.WithService(func(...testtypes.InterfaceA) testtypes.InterfaceB {
				return &testtypes.StructB{}
			}),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		LogError(t, err)

		assert.ErrorContains(t, err,
			"service func(...testtypes.InterfaceA) testtypes.InterfaceB: "+
				"dependency []testtypes.InterfaceA: "+
				"service func(testtypes.InterfaceD) testtypes.InterfaceA: "+
				"dependency testtypes.InterfaceD: service not registered",
		)
	})

	t.Run("registration aliases are validated once", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(
				func(testtypes.InterfaceD) *testtypes.StructA { return &testtypes.StructA{} },
				di.As[testtypes.InterfaceA](),
				di.As[*testtypes.StructA](),
			),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		LogError(t, err)

		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func(testtypes.InterfaceD) *testtypes.StructA: dependency testtypes.InterfaceD: service not registered",
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

	t.Run("only final variadic dependency is optional", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func([]testtypes.InterfaceC, ...testtypes.InterfaceA) testtypes.InterfaceB {
				return &testtypes.StructB{}
			}),
		)
		require.NoError(t, err)

		err = di.ValidateContainer(c)
		LogError(t, err)

		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func([]testtypes.InterfaceC, ...testtypes.InterfaceA) testtypes.InterfaceB: "+
			"dependency []testtypes.InterfaceC: service not registered",
		)
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

	t.Run("nested child validates scoped services from all ancestors", func(t *testing.T) {
		root, err := di.NewContainer(
			di.WithService(func(testtypes.InterfaceD) testtypes.InterfaceA {
				return &testtypes.StructA{}
			}, di.Scoped),
		)
		require.NoError(t, err)

		child, err := root.NewScope()
		require.NoError(t, err)
		grandchild, err := child.NewScope()
		require.NoError(t, err)

		err = di.ValidateContainer(grandchild)
		LogError(t, err)

		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func(testtypes.InterfaceD) testtypes.InterfaceA: "+
			"dependency testtypes.InterfaceD: service not registered",
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

		assert.EqualError(t, err, "di.ValidateContainer: "+
			"service func(testtypes.InterfaceB) testtypes.InterfaceC: "+
			"dependency testtypes.InterfaceB: service func(testtypes.InterfaceC) testtypes.InterfaceB: "+
			"dependency testtypes.InterfaceC: service func(testtypes.InterfaceB) testtypes.InterfaceC: "+
			"dependency cycle detected\n"+
			"service func(testtypes.InterfaceC) testtypes.InterfaceB: "+
			"dependency testtypes.InterfaceC: service func(testtypes.InterfaceB) testtypes.InterfaceC: "+
			"dependency testtypes.InterfaceB: service func(testtypes.InterfaceC) testtypes.InterfaceB: "+
			"dependency cycle detected",
		)
	})
}
