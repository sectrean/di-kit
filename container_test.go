package di_test

import (
	"context"
	"math/rand"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/internal/errors"
	"github.com/sectrean/di-kit/internal/mocks"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/sectrean/di-kit/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Add tests for the following:
// - more tests around the resolve locking

func Test_NewContainer(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		c, err := di.NewContainer()
		assert.NotNil(t, c)
		assert.NoError(t, err)
	})

	t.Run("WithService", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		assert.NotNil(t, c)
		assert.NoError(t, err)

		has := di.Contains[testtypes.InterfaceA](c)
		assert.True(t, has)
	})

	t.Run("WithService invalid type int", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(1234),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService int: invalid service type")
	})

	t.Run("WithService interface nil", func(t *testing.T) {
		var a testtypes.InterfaceA = nil
		c, err := di.NewContainer(
			di.WithService(a),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService: funcOrValue is nil")
	})

	t.Run("WithService pointer nil", func(t *testing.T) {
		var a *testtypes.StructA = nil
		c, err := di.NewContainer(
			di.WithService(a),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService: funcOrValue is nil")
	})

	t.Run("WithService invalid type di.Lifetime", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(di.SingletonLifetime, di.WithTag("tag")),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService di.Lifetime: invalid service type")
	})

	t.Run("WithService invalid type map", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(map[string]int{}),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService map[string]int: invalid service type")
	})

	t.Run("WithService invalid type *int", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() *int { return nil }),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() *int: invalid service type")
	})

	t.Run("WithService func returns unnamed func", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() func(http.Handler) http.Handler { return nil }),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() func(http.Handler) http.Handler: invalid service type")
	})

	t.Run("WithService invalid dependency type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(int) testtypes.InterfaceA { return nil }),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func(int) testtypes.InterfaceA: invalid dependency type int")
	})

	t.Run("WithService invalid dependency types", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(int, di.Lifetime) testtypes.InterfaceA { return nil }),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func(int, di.Lifetime) testtypes.InterfaceA: invalid dependency type int\n"+
			"invalid dependency type di.Lifetime")
	})

	t.Run("WithService As not assignable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.As[*testtypes.StructA]()),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() testtypes.InterfaceA: As *testtypes.StructA: type testtypes.InterfaceA not assignable to *testtypes.StructA")
	})

	t.Run("WithService As invalid service type map", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.CustomMap{}, di.As[map[string]any]()),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService testtypes.CustomMap: As map[string]interface {}: invalid service type")
	})

	t.Run("WithService SingletonLifetime value service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}, di.SingletonLifetime),
		)
		assert.NotNil(t, c)
		assert.NoError(t, err)
	})

	t.Run("WithService TransientLifetime value service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}, di.TransientLifetime),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService *testtypes.StructA: Lifetime Transient: invalid lifetime for value service")
	})

	t.Run("WithService As interface not assignable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}, di.As[testtypes.InterfaceB]()),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService *testtypes.StructA: As testtypes.InterfaceB: type *testtypes.StructA not assignable to testtypes.InterfaceB")
	})

	t.Run("WithService WithTagged parameter not found", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				di.WithTagged[testtypes.InterfaceB]("tag"),
			),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() testtypes.InterfaceA: WithTagged testtypes.InterfaceB: parameter not found")
	})

	t.Run("WithService UseCloseFunc not assignable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				di.UseCloseFunc(func(context.Context, *testtypes.StructA) error { return nil }),
			),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() testtypes.InterfaceA: UseCloseFunc: service type testtypes.InterfaceA is not assignable to *testtypes.StructA")
	})

	t.Run("WithService unsupported func signature", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() (testtypes.InterfaceA, testtypes.InterfaceB) { return nil, nil }),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err,
			"di.NewContainer: WithService func() (testtypes.InterfaceA, testtypes.InterfaceB): function must return Service or (Service, error)")
	})

	t.Run("WithService invalid type error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() error { return errors.New("test error") }),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() error: invalid service type")
	})

	t.Run("WithService invalid type context.Context", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(context.Background),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() context.Context: invalid service type")
	})

	t.Run("WithService invalid basic types", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService([]int{}),
			di.WithService(map[string]int{}),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService []int: invalid service type\n"+
			"WithService map[string]int: invalid service type",
		)
	})

	t.Run("multiple errors", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService([]testtypes.InterfaceA{}),
			di.WithService(testtypes.NewInterfaceA, di.As[testtypes.InterfaceB]()),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService []testtypes.InterfaceA: invalid service type\n"+
			"WithService func() testtypes.InterfaceA: As testtypes.InterfaceB: type testtypes.InterfaceA not assignable to testtypes.InterfaceB",
		)
	})

	t.Run("multiple service errors", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService([]testtypes.InterfaceA{}),
			di.WithService(testtypes.NewInterfaceA,
				di.As[testtypes.InterfaceB](),
				di.WithTagged[*testtypes.StructB]("tag"),
			),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService []testtypes.InterfaceA: invalid service type\n"+
			"WithService func() testtypes.InterfaceA: As testtypes.InterfaceB: type testtypes.InterfaceA not assignable to testtypes.InterfaceB\n"+
			"WithTagged *testtypes.StructB: parameter not found",
		)
	})

	t.Run("Module", func(t *testing.T) {
		c, err := di.NewContainer(
			di.Module{
				di.WithService(testtypes.NewInterfaceA),
				di.WithService(testtypes.NewInterfaceB),
			},
			di.WithService(testtypes.NewInterfaceC),
		)
		assert.NotNil(t, c)
		assert.NoError(t, err)
	})

	t.Run("WithModule WithService nil", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithModule([]di.ContainerOption{
				di.WithService(testtypes.NewInterfaceA),
				di.WithService(nil),
			}),
			di.WithService(testtypes.NewInterfaceC),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService: funcOrValue is nil")
	})

	t.Run("WithDependencyValidation", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB),
			di.WithDependencyValidation(),
		)
		assert.NotNil(t, c)
		assert.NoError(t, err)
	})

	t.Run("WithDependencyValidation invalid service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceB),
			di.WithDependencyValidation(),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err,
			"di.NewContainer: WithDependencyValidation: service testtypes.InterfaceB: dependency testtypes.InterfaceA: service not registered")
	})

	t.Run("WithDependencyValidation scoped service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceC, di.ScopedLifetime),
			di.WithDependencyValidation(),
		)
		assert.NotNil(t, c)
		assert.NoError(t, err)
	})

	t.Run("WithDependencyValidation dependency cycle", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func(context.Context, testtypes.InterfaceC) testtypes.InterfaceB { return nil }),
			di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceC { return nil }),
			di.WithDependencyValidation(),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		// The exact error message is non-deterministic because it depends on map iteration order
		assert.ErrorContains(t, err, "dependency cycle detected")
	})

	t.Run("WithDependencyValidation dependency cycle single type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(context.Context, testtypes.InterfaceA) testtypes.InterfaceA { return nil }),
			di.WithDependencyValidation(),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithDependencyValidation: service testtypes.InterfaceA: dependency testtypes.InterfaceA: dependency cycle detected")
	})

	t.Run("WithDependencyValidation slice dependency", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func([]testtypes.InterfaceA) testtypes.InterfaceB {
				return testtypes.StructB{}
			}),
			di.WithService(func([]testtypes.InterfaceC) testtypes.InterfaceD {
				return testtypes.StructD{}
			}),
			di.WithDependencyValidation(),
		)
		testutils.LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err,
			"di.NewContainer: WithDependencyValidation: service testtypes.InterfaceD: dependency testtypes.InterfaceC: service not registered",
		)
	})

	t.Run("WithDependencyValidation variadic dependency", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func(...testtypes.InterfaceA) testtypes.InterfaceB {
				return testtypes.StructB{}
			}),
			di.WithService(func(...testtypes.InterfaceC) testtypes.InterfaceD {
				return testtypes.StructD{}
			}),
			di.WithDependencyValidation(),
		)
		assert.NotNil(t, c)
		assert.NoError(t, err)
	})
}

func Test_Container_NewScope(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.ScopedLifetime),
		)
		require.NoError(t, err)

		scope, err := c.NewScope()
		assert.NotNil(t, scope)
		assert.NoError(t, err)

		assert.True(t, di.Contains[testtypes.InterfaceA](scope))
		assert.True(t, di.Contains[testtypes.InterfaceB](scope))
	})

	t.Run("WithService", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(testtypes.NewInterfaceB),
		)
		assert.NotNil(t, scope)
		assert.NoError(t, err)

		assert.True(t, di.Contains[testtypes.InterfaceA](c))
		assert.False(t, di.Contains[testtypes.InterfaceB](c))

		assert.True(t, di.Contains[testtypes.InterfaceA](scope))
		assert.True(t, di.Contains[testtypes.InterfaceB](scope))
	})

	t.Run("WithService invalid type di.Lifetime", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(di.ScopedLifetime),
		)
		testutils.LogError(t, err)

		assert.Nil(t, scope)
		assert.EqualError(t, err, "di.Container.NewScope: WithService di.Lifetime: invalid service type")
	})

	t.Run("parent closed", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		err = c.Close(ctx)
		assert.NoError(t, err)

		scope, err := c.NewScope()
		testutils.LogError(t, err)

		assert.Nil(t, scope)
		assert.EqualError(t, err, "di.Container.NewScope: container closed")
	})

	t.Run("WithDependencyValidation", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.ScopedLifetime),
			di.WithDependencyValidation(),
		)
		require.NoError(t, err)

		_, err = c.NewScope(
			di.WithDependencyValidation(),
		)
		assert.NoError(t, err)
	})

	t.Run("WithDependencyValidation service not registered", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceB, di.ScopedLifetime),
			di.WithDependencyValidation(),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithDependencyValidation(),
		)
		testutils.LogError(t, err)

		assert.Nil(t, scope)
		assert.EqualError(t, err, "di.Container.NewScope: WithDependencyValidation: service testtypes.InterfaceB: dependency testtypes.InterfaceA: service not registered")
	})

	t.Run("WithDependencyValidation dependency cycle", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceC { return nil }, di.ScopedLifetime),
			di.WithDependencyValidation(),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(func(testtypes.InterfaceC) testtypes.InterfaceB { return nil }),
			di.WithDependencyValidation(),
		)
		testutils.LogError(t, err)

		assert.Nil(t, scope)
		// The exact error message is non-deterministic because it depends on map iteration order
		assert.ErrorContains(t, err, "dependency cycle detected")
	})
}

func Test_Container_Contains(t *testing.T) {
	t.Run("service registered", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		has := c.Contains(reflect.TypeFor[testtypes.InterfaceA]())
		assert.True(t, has)
	})

	t.Run("service not registered", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		has := c.Contains(reflect.TypeFor[testtypes.InterfaceA]())
		assert.False(t, has)
	})

	t.Run("WithTag", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithTag("tag")),
		)
		require.NoError(t, err)

		has := c.Contains(reflect.TypeFor[testtypes.InterfaceA](), di.WithTag("tag"))
		assert.True(t, has)

		has = c.Contains(reflect.TypeFor[testtypes.InterfaceA]())
		assert.False(t, has)

		has = c.Contains(reflect.TypeFor[testtypes.InterfaceA](), di.WithTag("other"))
		assert.False(t, has)
	})

	t.Run("found in parent scope", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(testtypes.NewInterfaceB),
		)
		require.NoError(t, err)

		has := scope.Contains(reflect.TypeFor[testtypes.InterfaceA]())
		assert.True(t, has)

		has = scope.Contains(reflect.TypeFor[testtypes.InterfaceB]())
		assert.True(t, has)
	})

	t.Run("slice service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		has := c.Contains(reflect.TypeFor[[]testtypes.InterfaceA]())
		assert.True(t, has)
	})

	t.Run("slice service not registered", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		has := c.Contains(reflect.TypeFor[[]testtypes.InterfaceB]())
		assert.False(t, has)
	})

	t.Run("WithTag slice service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithTag(1)),
		)
		require.NoError(t, err)

		has := c.Contains(reflect.TypeFor[[]testtypes.InterfaceA]())
		assert.False(t, has)

		has = c.Contains(reflect.TypeFor[[]testtypes.InterfaceA](), di.WithTag(1))
		assert.True(t, has)
	})
}

func Test_Container_Resolve(t *testing.T) {
	t.Run("value service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[*testtypes.StructA](ctx, c)

		assert.Equal(t, &testtypes.StructA{}, got)
		assert.NoError(t, err)
	})

	t.Run("value service from child scope", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}),
		)
		require.NoError(t, err)

		scope, err := c.NewScope()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[*testtypes.StructA](ctx, scope)

		assert.Equal(t, &testtypes.StructA{}, got)
		assert.NoError(t, err)
	})

	t.Run("func interface nil", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				return nil
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)

		assert.Nil(t, got)
		assert.True(t, got == nil)
		assert.NoError(t, err)
	})

	t.Run("func interface typed nil", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				var a *testtypes.StructA = nil
				return a
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)

		assert.Nil(t, got)
		assert.True(t, got == nil)
		assert.NoError(t, err)
	})

	t.Run("func pointer nil", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() *testtypes.StructA {
				return nil
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[*testtypes.StructA](ctx, c)

		assert.Nil(t, got)
		assert.True(t, got == nil)
		assert.NoError(t, err)
	})

	t.Run("func error nil", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() (testtypes.InterfaceA, error) {
				return &testtypes.StructA{}, nil
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)

		assert.Equal(t, &testtypes.StructA{}, got)
		assert.True(t, err == nil)
		assert.NoError(t, err)
	})

	t.Run("func error typed nil", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() (testtypes.InterfaceA, error) {
				var svcErr *testtypes.CustomError = nil
				return &testtypes.StructA{}, svcErr
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)

		assert.Equal(t, &testtypes.StructA{}, got)
		assert.True(t, err == nil)
		assert.NoError(t, err)
	})

	t.Run("value struct", func(t *testing.T) {
		a1 := testtypes.StructA{Tag: 1}

		c, err := di.NewContainer(
			di.WithService(a1),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.StructA](ctx, c)
		assert.Equal(t, a1, got)
		assert.NoError(t, err)
	})

	t.Run("func struct", func(t *testing.T) {
		a1 := testtypes.StructA{Tag: 1}

		c, err := di.NewContainer(
			di.WithService(func() testtypes.StructA { return a1 }),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.StructA](ctx, c)
		assert.Equal(t, a1, got)
		assert.NoError(t, err)
	})

	t.Run("named basic types", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.CustomString("test")),
			di.WithService(func(s testtypes.CustomString) testtypes.CustomStringCollection {
				return testtypes.CustomStringCollection{
					string(s) + "1",
					string(s) + "2",
				}
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.CustomStringCollection](ctx, c)

		assert.NoError(t, err)
		assert.Equal(t, testtypes.CustomStringCollection{"test1", "test2"}, got)
	})

	t.Run("func service named func type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewMiddleware),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.HTTPMiddleware](ctx, c)

		assert.NotNil(t, got)
		assert.NoError(t, err)
	})

	t.Run("value service named func type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewMiddleware()),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.HTTPMiddleware](ctx, c)

		assert.NotNil(t, got)
		assert.NoError(t, err)
	})

	t.Run("As func pointer nil", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() *testtypes.StructA { return nil },
				di.As[testtypes.InterfaceA](),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)

		assert.Nil(t, got)
		assert.True(t, got == nil)
		assert.NoError(t, err)
	})

	t.Run("func no deps", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)

		assert.Equal(t, &testtypes.StructA{}, got)
		assert.NoError(t, err)
	})

	t.Run("container closed", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		err = c.Close(ctx)
		assert.NoError(t, err)

		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.True(t, got == nil)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: container closed")
	})

	t.Run("context canceled", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: context canceled")
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("context deadline exceeded", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), -1)
		defer cancel()

		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: context deadline exceeded")
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("not registered service", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: service not registered")
	})

	t.Run("not registered di.Scope", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[di.Scope](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve di.Scope: service not registered")
	})

	t.Run("not registered context.Context", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[context.Context](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve context.Context: service not registered")
	})

	t.Run("dependency not registered", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceB),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: dependency testtypes.InterfaceA: service not registered")
	})

	t.Run("dependency cycle", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceA { return nil }),
			di.WithService(testtypes.NewInterfaceB),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: dependency testtypes.InterfaceB: dependency testtypes.InterfaceA: dependency cycle detected")
	})

	t.Run("SingletonLifetime", func(t *testing.T) {
		calls := 0

		c, err := di.NewContainer(
			di.WithService(
				func() testtypes.InterfaceA {
					calls++
					return &testtypes.StructA{Tag: 1}
				},
				di.SingletonLifetime,
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		a1, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Equal(t, &testtypes.StructA{Tag: 1}, a1)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)

		a2, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Same(t, a1, a2)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("SingletonLifetime from child scope", func(t *testing.T) {
		calls := 0

		c, err := di.NewContainer(
			di.WithService(
				func() testtypes.InterfaceA {
					calls++
					return &testtypes.StructA{Tag: 1}
				},
				di.SingletonLifetime,
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		a1, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Equal(t, &testtypes.StructA{Tag: 1}, a1)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)

		scope, err := c.NewScope()
		require.NoError(t, err)

		a2, err := di.Resolve[testtypes.InterfaceA](ctx, scope)
		assert.Same(t, a1, a2)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("TransientLifetime", func(t *testing.T) {
		calls := 0

		c, err := di.NewContainer(
			di.WithService(
				func() testtypes.InterfaceA {
					calls++
					return &testtypes.StructA{Tag: calls}
				},
				di.TransientLifetime,
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		a1, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Equal(t, &testtypes.StructA{Tag: 1}, a1)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)

		a2, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Equal(t, &testtypes.StructA{Tag: 2}, a2)
		assert.NoError(t, err)
		assert.Equal(t, 2, calls)
	})

	t.Run("ScopedLifetime", func(t *testing.T) {
		calls := 0

		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(
				func(a testtypes.InterfaceA) testtypes.InterfaceB {
					calls++
					assert.NotNil(t, a)
					return &testtypes.StructB{}
				},
				di.ScopedLifetime,
			),
		)
		require.NoError(t, err)

		ctx := context.Background()

		for range 3 {
			scope, err := c.NewScope()
			require.NoError(t, err)

			b1, err := di.Resolve[testtypes.InterfaceB](ctx, scope)
			assert.NotNil(t, b1)
			assert.NoError(t, err)

			b2, err := di.Resolve[testtypes.InterfaceB](ctx, scope)
			assert.NotNil(t, b2)
			assert.NoError(t, err)

			assert.Exactly(t, b1, b2)
		}

		assert.Equal(t, 3, calls)
	})

	t.Run("ScopedLifetime resolve from root", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.ScopedLifetime),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: scoped service must be resolved from a child scope")
	})

	t.Run("ScopedLifetime multi level", func(t *testing.T) {
		root, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		scope1, err := root.NewScope(
			di.WithService(testtypes.NewInterfaceB, di.ScopedLifetime),
		)
		require.NoError(t, err)

		ctx := context.Background()
		b, err := di.Resolve[testtypes.InterfaceB](ctx, scope1)
		testutils.LogError(t, err)

		assert.Nil(t, b)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: scoped service must be resolved from a child scope")

		scope2, err := scope1.NewScope()
		require.NoError(t, err)

		b, err = di.Resolve[testtypes.InterfaceB](ctx, scope2)
		assert.NotNil(t, b)
		assert.NoError(t, err)
	})

	t.Run("ScopedLifetime dependencies", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.SingletonLifetime),
			di.WithService(testtypes.NewInterfaceC, di.ScopedLifetime),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(testtypes.NewInterfaceB),
			di.WithService(testtypes.NewInterfaceD),
		)
		require.NoError(t, err)

		ctx := context.Background()
		gotC, err := di.Resolve[testtypes.InterfaceC](ctx, scope)
		assert.NotNil(t, gotC)
		assert.NoError(t, err)

		gotD, err := di.Resolve[testtypes.InterfaceD](ctx, scope)
		assert.NotNil(t, gotD)
		assert.NoError(t, err)
	})

	t.Run("ScopedLifetime captive dependency", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.ScopedLifetime),
			di.WithService(testtypes.NewInterfaceB, di.SingletonLifetime),
		)
		require.NoError(t, err)

		ctx := context.Background()
		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, b)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: dependency testtypes.InterfaceA: scoped service must be resolved from a child scope")
	})

	t.Run("slice service", func(t *testing.T) {
		f := &testtypes.Factory{}

		c, err := di.NewContainer(
			di.WithService(f.NewInterfaceA),
			di.WithService(f.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[[]testtypes.InterfaceA](ctx, c)

		want := testtypes.ExpectInterfaceA(2)
		assert.ElementsMatch(t, want, got)
		assert.NoError(t, err)
	})

	t.Run("slice service values", func(t *testing.T) {
		f := &testtypes.Factory{}

		c, err := di.NewContainer(
			di.WithService(f.NewStructA()),
			di.WithService(f.NewStructA()),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[[]*testtypes.StructA](ctx, c)

		want := testtypes.ExpectStructA(2)
		assert.ElementsMatch(t, want, got)
		assert.NoError(t, err)
	})

	t.Run("slice service dependency", func(t *testing.T) {
		f := &testtypes.Factory{}

		c, err := di.NewContainer(
			di.WithService(f.NewInterfaceA),
			di.WithService(f.NewInterfaceA),
			di.WithService(func(aa []testtypes.InterfaceA) testtypes.InterfaceB {
				assert.Equal(t, testtypes.ExpectInterfaceA(2), aa)
				return &testtypes.StructB{}
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		assert.NotNil(t, b)
		assert.NoError(t, err)
	})

	t.Run("slice service of one", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func([]testtypes.InterfaceA) testtypes.InterfaceB {
				return &testtypes.StructB{}
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		aa, err := di.Resolve[[]testtypes.InterfaceA](ctx, c)
		assert.Equal(t, []testtypes.InterfaceA{&testtypes.StructA{}}, aa)
		assert.NoError(t, err)

		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		assert.NotNil(t, b)
		assert.NoError(t, err)
	})

	t.Run("slice service variadic", func(t *testing.T) {
		f := &testtypes.Factory{}
		want := testtypes.ExpectInterfaceA(2)

		c, err := di.NewContainer(
			di.WithService(f.NewInterfaceA),
			di.WithService(f.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB),
			di.WithService(func(b testtypes.InterfaceB, aa ...testtypes.InterfaceA) testtypes.InterfaceD {
				assert.Equal(t, &testtypes.StructB{}, b)
				assert.ElementsMatch(t, want, aa)
				return &testtypes.StructD{}
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		d, err := di.Resolve[testtypes.InterfaceD](ctx, c)
		assert.Equal(t, &testtypes.StructD{}, d)
		assert.NoError(t, err)
	})

	t.Run("slice service variadic optional", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(aa ...testtypes.InterfaceA) testtypes.InterfaceB {
				assert.Empty(t, aa)
				return &testtypes.StructB{}
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		d, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		assert.Equal(t, &testtypes.StructB{}, d)
		assert.NoError(t, err)
	})

	t.Run("slice service across scopes", func(t *testing.T) {
		f := &testtypes.Factory{}

		c, err := di.NewContainer(
			di.WithService(f.NewStructA),
			di.WithService(f.NewStructA),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(f.NewStructA),
			di.WithService(f.NewStructA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[[]*testtypes.StructA](ctx, scope)

		want := testtypes.ExpectStructA(4)
		assert.ElementsMatch(t, want, got)
		assert.NoError(t, err)
	})

	t.Run("slice service error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func() (testtypes.InterfaceA, error) {
				return nil, errors.New("test error")
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[[]testtypes.InterfaceA](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve []testtypes.InterfaceA: test error")
	})

	t.Run("WithTag slice service", func(t *testing.T) {
		f := &testtypes.Factory{}

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				assert.Fail(t, "should not be called")
				return &testtypes.StructA{}
			}),
			di.WithService(f.NewInterfaceA, di.WithTag(1)),
			di.WithService(f.NewInterfaceA, di.WithTag(1)),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[[]testtypes.InterfaceA](ctx, c, di.WithTag(1))

		want := testtypes.ExpectInterfaceA(2)
		assert.Equal(t, want, got)
		assert.NoError(t, err)
	})

	t.Run("slice service nil service", func(t *testing.T) {
		a1 := &testtypes.StructA{Tag: 1}

		c, err := di.NewContainer(
			di.WithService(func() *testtypes.StructA {
				return nil
			}),
			di.WithService(a1),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[[]*testtypes.StructA](ctx, c)

		want := []*testtypes.StructA{a1}
		assert.Equal(t, want, got, "nil service should not be included in the slice")
		assert.NoError(t, err)
	})

	t.Run("slice service not registered", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[[]testtypes.InterfaceA](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve []testtypes.InterfaceA: service not registered")
	})

	t.Run("WithTag slice service not registered", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[[]testtypes.InterfaceA](ctx, c, di.WithTag(1))
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve []testtypes.InterfaceA (Tag 1): service not registered")
	})

	t.Run("As", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{},
				di.As[testtypes.InterfaceA](),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Equal(t, &testtypes.StructA{}, got)
		assert.NoError(t, err)

		got, err = di.Resolve[*testtypes.StructA](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, got)
	})

	t.Run("As original type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(
				func() *testtypes.StructA {
					return &testtypes.StructA{Tag: 1}
				},
				di.As[testtypes.InterfaceA](),
				di.As[*testtypes.StructA](),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		a1, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.NotNil(t, a1)
		assert.NoError(t, err)

		a2, err := di.Resolve[*testtypes.StructA](ctx, c)
		assert.NotNil(t, a2)
		assert.NoError(t, err)

		assert.Same(t, a1, a2)
	})

	t.Run("WithTag func service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(
				func() testtypes.InterfaceA {
					return &testtypes.StructA{Tag: 1}
				},
				di.WithTag("tag"),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		a1, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithTag("tag"))
		assert.NotNil(t, a1)
		assert.NoError(t, err)

		a2, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Nil(t, a2)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: service not registered")
	})

	t.Run("WithTag value service", func(t *testing.T) {
		a := &testtypes.StructA{Tag: 1}

		c, err := di.NewContainer(
			di.WithService(a, di.WithTag("tag")),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[*testtypes.StructA](ctx, c, di.WithTag("tag"))
		assert.Same(t, a, got)
		assert.NoError(t, err)
	})

	t.Run("WithTag interface", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewStructAPtr,
				di.As[testtypes.InterfaceA](),
				di.WithTag("tag"),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithTag("tag"))
		assert.NotNil(t, got)
		assert.NoError(t, err)

		got, err = di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: service not registered")
	})

	t.Run("WithTag mixed", func(t *testing.T) {
		a1 := &testtypes.StructA{Tag: 1}
		a2 := &testtypes.StructA{Tag: 2}

		c, err := di.NewContainer(
			di.WithService(a1, di.As[testtypes.InterfaceA]()),
			di.WithService(a2, di.As[testtypes.InterfaceA](), di.WithTag(2)),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Same(t, a1, got)
		assert.NoError(t, err)

		got, err = di.Resolve[testtypes.InterfaceA](ctx, c, di.WithTag(2))
		assert.Same(t, a2, got)
		assert.NoError(t, err)
	})

	t.Run("WithTag not registered", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithTag("tag")),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithTag("other"))
		testutils.LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA (Tag other): service not registered")
	})

	t.Run("WithTagged", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				di.WithTag("A1"),
			),
			di.WithService(func() (testtypes.InterfaceA, error) {
				assert.Fail(t, "should not be called")
				return &testtypes.StructA{}, nil
			}),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceB {
				return &testtypes.StructB{}
			}, di.WithTagged[testtypes.InterfaceA]("A1")),
		)
		require.NoError(t, err)

		ctx := context.Background()

		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		assert.Equal(t, &testtypes.StructB{}, b)
		assert.NoError(t, err)
	})

	t.Run("WithTagged decorator", func(t *testing.T) {
		a1 := &testtypes.StructA{Tag: 1}
		a2 := &testtypes.StructA{Tag: 2}

		c, err := di.NewContainer(
			di.WithService(func(a testtypes.InterfaceA) testtypes.InterfaceA {
				assert.Same(t, a1, a)
				return a2
			}, di.WithTagged[testtypes.InterfaceA]("decorate me")),
			di.WithService(a1, di.As[testtypes.InterfaceA](), di.WithTag("decorate me")),
		)
		require.NoError(t, err)

		ctx := context.Background()

		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Same(t, a2, got)
		assert.NoError(t, err)
	})

	t.Run("WithTagged multiple", func(t *testing.T) {
		a1 := &testtypes.StructA{Tag: 1}
		a2 := &testtypes.StructA{Tag: 2}

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return a1 }, di.WithTag("1")),
			di.WithService(func() testtypes.InterfaceA { return a2 }, di.WithTag("2")),
			di.WithService(
				func(aa2 testtypes.InterfaceA, aa1 testtypes.InterfaceA) testtypes.InterfaceB {
					assert.Same(t, a1, aa1)
					assert.Same(t, a2, aa2)
					return &testtypes.StructB{}
				},
				di.WithTagged[testtypes.InterfaceA]("2"),
				di.WithTagged[testtypes.InterfaceA]("1"),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		assert.Equal(t, &testtypes.StructB{}, got)
		assert.NoError(t, err)
	})

	t.Run("func error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() (testtypes.InterfaceA, error) {
				return nil, errors.New("constructor error")
			}),
			di.WithService(testtypes.NewInterfaceB),
		)
		require.NoError(t, err)

		ctx := context.Background()
		a, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, a)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: constructor error")

		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		testutils.LogError(t, err)

		assert.Nil(t, b)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: dependency testtypes.InterfaceA: constructor error")
	})

	t.Run("dependency nil", func(t *testing.T) {
		calls := 0
		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return nil }),
			di.WithService(func(a testtypes.InterfaceA) testtypes.InterfaceB {
				calls++
				assert.Nil(t, a)
				return &testtypes.StructB{}
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		assert.NotNil(t, b)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("dependency context.Context", func(t *testing.T) {
		ctx := testutils.ContextWithTestValue(context.Background(), "value")

		c, err := di.NewContainer(
			di.WithService(func(ctxDep context.Context) testtypes.InterfaceA {
				assert.Same(t, ctx, ctxDep)
				return &testtypes.StructA{}
			}),
		)
		require.NoError(t, err)

		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.NotNil(t, got)
		assert.NoError(t, err)
	})

	t.Run("dependency di.Scope", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func(ctx context.Context, scope di.Scope) *ScopeFactory[testtypes.InterfaceA] {
				// We cannot call Resolve on the scope here.
				a, err := di.Resolve[testtypes.InterfaceA](ctx, scope)
				testutils.LogError(t, err)

				assert.Nil(t, a)
				assert.EqualError(t, err,
					"di.Container.Resolve testtypes.InterfaceA: "+
						"not supported within service constructor function")

				// Contains can be called though
				assert.True(t, di.Contains[testtypes.InterfaceA](scope))

				// We have to store it and we can call Resolve later.
				return NewScopeFactory(scope, func(ctx context.Context, s di.Scope) (testtypes.InterfaceA, error) {
					return di.Resolve[testtypes.InterfaceA](ctx, s)
				})
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		factory, err := di.Resolve[*ScopeFactory[testtypes.InterfaceA]](ctx, c)
		require.NoError(t, err)

		a, err := factory.Build(ctx)
		assert.NotNil(t, a)
		assert.NoError(t, err)
	})

	t.Run("WithModule", func(t *testing.T) {
		// The module service should be registered first since the module is added before the
		// other service registrations.
		a1 := &testtypes.StructA{Tag: 1}
		a2 := &testtypes.StructA{Tag: 2}

		c, err := di.NewContainer(
			di.WithModule(di.Module{
				di.WithService(a1, di.As[testtypes.InterfaceA]()),
				di.WithService(testtypes.NewInterfaceB),
			}),
			di.WithService(testtypes.NewInterfaceC),
			di.WithService(a2, di.As[testtypes.InterfaceA]()),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceC](ctx, c)
		assert.NotNil(t, got)
		assert.NoError(t, err)

		aGot, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Same(t, a2, aGot)
		assert.NoError(t, err)
	})

	// Concurrent tests should be run with the -race flag to check for race conditions

	t.Run("concurrent", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB),
			di.WithService(testtypes.NewInterfaceC),
			di.WithService(testtypes.NewInterfaceD),
		)
		require.NoError(t, err)

		ctx := context.Background()
		wg := sync.WaitGroup{}
		wg.Add(4)

		go func() {
			defer wg.Done()

			a, err := di.Resolve[testtypes.InterfaceA](ctx, c)
			assert.NotNil(t, a)
			assert.NoError(t, err)
		}()
		go func() {
			defer wg.Done()

			b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
			assert.NotNil(t, b)
			assert.NoError(t, err)
		}()
		go func() {
			defer wg.Done()

			c, err := di.Resolve[testtypes.InterfaceC](ctx, c)
			assert.NotNil(t, c)
			assert.NoError(t, err)
		}()
		go func() {
			defer wg.Done()

			d, err := di.Resolve[testtypes.InterfaceD](ctx, c)
			assert.NotNil(t, d)
			assert.NoError(t, err)
		}()

		wg.Wait()
	})

	t.Run("concurrent singleton", func(t *testing.T) {
		expected := &testtypes.StructA{Tag: 1}
		calls := 0

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				calls++
				return expected
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		wg := sync.WaitGroup{}

		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()

				got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
				assert.Same(t, expected, got)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()
		assert.Equal(t, 1, calls)
	})

	t.Run("concurrent scoped", func(t *testing.T) {
		r := rand.Intn(10)
		calls := 0

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				time.Sleep(time.Duration(r) * time.Microsecond)
				return &testtypes.StructA{}
			}, di.TransientLifetime),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceB {
				calls++
				return &testtypes.StructB{}
			}, di.ScopedLifetime),
		)
		require.NoError(t, err)

		scope, err := c.NewScope()
		require.NoError(t, err)

		ctx := context.Background()
		wg := sync.WaitGroup{}

		for range 100 {
			wg.Add(1)
			go func() {
				defer wg.Done()

				got, err := di.Resolve[testtypes.InterfaceB](ctx, scope)
				assert.NotNil(t, got)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()
		assert.Equal(t, 1, calls)
	})

	t.Run("concurrent singleton race", func(t *testing.T) {
		wait := make(chan struct{})
		calls := 0

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				close(wait)
				return &testtypes.StructA{}
			}),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceB {
				calls++
				return &testtypes.StructB{}
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		wg := sync.WaitGroup{}
		wg.Add(2)

		go func() {
			defer wg.Done()

			_, err := di.Resolve[testtypes.InterfaceB](ctx, c)
			assert.NoError(t, err)
		}()

		go func() {
			defer wg.Done()

			// Wait until the first goroutine has started resolving
			<-wait

			_, err := di.Resolve[testtypes.InterfaceB](ctx, c)
			assert.NoError(t, err)
		}()

		wg.Wait()
		assert.Equal(t, 1, calls)
	})

	t.Run("concurrent dependency cycle", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceB {
				assert.Fail(t, "constructor func should not get called")
				return nil
			}),
			di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceA {
				assert.Fail(t, "constructor func should not get called")
				return nil
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		wg := sync.WaitGroup{}
		wg.Add(2)

		go func() {
			defer wg.Done()
			_, err := di.Resolve[testtypes.InterfaceA](ctx, c)
			testutils.LogError(t, err)
			assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: dependency testtypes.InterfaceB: dependency testtypes.InterfaceA: dependency cycle detected")
		}()

		go func() {
			defer wg.Done()
			_, err := di.Resolve[testtypes.InterfaceB](ctx, c)
			testutils.LogError(t, err)
			assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: dependency testtypes.InterfaceA: dependency testtypes.InterfaceB: dependency cycle detected")
		}()

		wg.Wait()
	})

	t.Run("concurrent dependency cycle workaround", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func(a testtypes.InterfaceA, c testtypes.InterfaceC) testtypes.InterfaceB {
				assert.NotNil(t, a)
				assert.NotNil(t, c)
				return &testtypes.StructB{}
			}),
			di.WithService(func(scope di.Scope) testtypes.InterfaceC {
				c := mocks.NewInterfaceCMock(t)

				// The circular dependency can be resolved by using a scope
				c.EXPECT().C().Run(func() {
					b, err := di.Resolve[testtypes.InterfaceB](context.Background(), scope)
					assert.NotNil(t, b)
					assert.NoError(t, err)
				})

				return c
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		wg := sync.WaitGroup{}
		wg.Add(2)

		go func() {
			defer wg.Done()

			b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
			assert.NotNil(t, b)
			assert.NoError(t, err)
		}()

		go func() {
			defer wg.Done()

			c, err := di.Resolve[testtypes.InterfaceC](ctx, c)
			assert.NotNil(t, c)
			assert.NoError(t, err)

			c.C()
		}()

		wg.Wait()
	})
}

func Test_Container_Close(t *testing.T) {
	t.Run("already closed", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		err = c.Close(ctx)
		assert.NoError(t, err)

		err = c.Close(ctx)
		testutils.LogError(t, err)

		assert.EqualError(t, err, "di.Container.Close: closed already: container closed")
	})

	t.Run("all close funcs", func(t *testing.T) {
		ctx := testutils.ContextWithTestValue(context.Background(), "value")

		aMock := mocks.NewInterfaceAMock(t)
		aMock.EXPECT().
			Close(ctx).
			Return(nil).
			Once()
		bMock := mocks.NewInterfaceBMock(t)
		bMock.EXPECT().
			Close(ctx).
			Once()
		cMock := mocks.NewInterfaceCMock(t)
		cMock.EXPECT().
			Close().
			Return(nil).
			Once()
		dMock := mocks.NewInterfaceDMock(t)
		dMock.EXPECT().
			Close().
			Once()

		scope, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return aMock }),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceB { return bMock }),
			di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceC { return cMock }),
			di.WithService(func(testtypes.InterfaceC) testtypes.InterfaceD { return dMock }),
		)
		require.NoError(t, err)

		_, err = di.Resolve[testtypes.InterfaceD](ctx, scope)
		assert.NoError(t, err)

		err = scope.Close(ctx)
		assert.NoError(t, err)
	})

	t.Run("dependency sequence", func(t *testing.T) {
		calls := 0
		ctx := context.Background()

		aMock := mocks.NewInterfaceAMock(t)
		aMock.EXPECT().
			Close(ctx).
			RunAndReturn(func(context.Context) error {
				assert.Equal(t, 1, calls, "a should be closed after b")
				calls++
				return nil
			}).
			Once()
		bMock := mocks.NewInterfaceBMock(t)
		bMock.EXPECT().
			Close(ctx).
			Run(func(context.Context) {
				assert.Equal(t, 0, calls, "b should be closed before a")
				calls++
			}).
			Once()

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return aMock }),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceB { return bMock }),
		)
		require.NoError(t, err)

		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		assert.NotNil(t, b)
		assert.NoError(t, err)

		// b doesn't have any dependencies so it should get closed first

		err = c.Close(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 2, calls)
	})

	t.Run("func not resolved", func(t *testing.T) {
		aMock := mocks.NewInterfaceAMock(t)

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return aMock }),
		)
		require.NoError(t, err)

		ctx := context.Background()
		err = c.Close(ctx)
		assert.NoError(t, err)
	})

	t.Run("value resolved", func(t *testing.T) {
		aMock := mocks.NewInterfaceAMock(t)

		c, err := di.NewContainer(
			di.WithService(aMock, di.As[testtypes.InterfaceA]()),
		)
		require.NoError(t, err)

		ctx := context.Background()
		a, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.NotNil(t, a)
		assert.NoError(t, err)

		err = c.Close(ctx)
		assert.NoError(t, err)
	})

	t.Run("value not resolved", func(t *testing.T) {
		aMock := mocks.NewInterfaceAMock(t)

		c, err := di.NewContainer(
			di.WithService(aMock),
		)
		require.NoError(t, err)

		ctx := context.Background()
		err = c.Close(ctx)
		assert.NoError(t, err)
	})

	t.Run("closer error", func(t *testing.T) {
		ctx := context.Background()

		aMock := mocks.NewInterfaceAMock(t)
		aMock.EXPECT().
			Close(ctx).
			Return(errors.New("err a")).
			Once()
		cMock := mocks.NewInterfaceCMock(t)
		cMock.EXPECT().
			Close().
			Return(nil).
			Once()

		scope, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return aMock }),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceC { return cMock }),
		)
		require.NoError(t, err)

		_, err = di.Resolve[testtypes.InterfaceC](ctx, scope)
		assert.NoError(t, err)

		err = scope.Close(ctx)
		testutils.LogError(t, err)
		assert.EqualError(t, err, "di.Container.Close: err a")
	})

	t.Run("closer errors", func(t *testing.T) {
		ctx := context.Background()

		aMock := mocks.NewInterfaceAMock(t)
		aMock.EXPECT().
			Close(ctx).
			Return(errors.New("err a")).
			Once()
		cMock := mocks.NewInterfaceCMock(t)
		cMock.EXPECT().
			Close().
			Return(errors.New("err c")).
			Once()

		scope, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return aMock }),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceC { return cMock }),
		)
		require.NoError(t, err)

		_, err = di.Resolve[testtypes.InterfaceC](ctx, scope)
		assert.NoError(t, err)

		err = scope.Close(ctx)
		testutils.LogError(t, err)
		assert.EqualError(t, err, "di.Container.Close: err c\nerr a")
	})

	t.Run("IgnoreCloser func service", func(t *testing.T) {
		aMock := mocks.NewInterfaceAMock(t)

		scope, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return aMock },
				di.IgnoreCloser(),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		_, err = di.Resolve[testtypes.InterfaceA](ctx, scope)
		assert.NoError(t, err)

		err = scope.Close(ctx)
		assert.NoError(t, err)
	})

	t.Run("UseCloser value service", func(t *testing.T) {
		ctx := context.Background()

		aMock := mocks.NewInterfaceAMock(t)
		aMock.EXPECT().
			Close(ctx).
			Return(nil).
			Once()

		c, err := di.NewContainer(
			di.WithService(aMock,
				di.As[testtypes.InterfaceA](),
				di.UseCloser(),
			),
		)
		require.NoError(t, err)

		// Value service should be close even if it is never resolved
		err = c.Close(ctx)
		assert.NoError(t, err)
	})

	t.Run("UseCloseFunc func service", func(t *testing.T) {
		ctx := context.Background()

		aMock := mocks.NewInterfaceAMock(t)
		aClosed := false

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return aMock },
				di.UseCloseFunc(func(context.Context, testtypes.InterfaceA) error {
					aClosed = true
					return nil
				}),
			),
		)
		require.NoError(t, err)

		_, err = di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.NoError(t, err)

		err = c.Close(ctx)
		assert.NoError(t, err)

		assert.True(t, aClosed)
	})

	t.Run("UseCloseFunc value service", func(t *testing.T) {
		ctx := context.Background()

		aMock := mocks.NewInterfaceAMock(t)
		aClosed := false

		c, err := di.NewContainer(
			di.WithService(aMock,
				di.As[testtypes.InterfaceA](),
				di.UseCloseFunc(func(context.Context, testtypes.InterfaceA) error {
					aClosed = true
					return nil
				}),
			),
		)
		require.NoError(t, err)

		err = c.Close(ctx)
		assert.NoError(t, err)

		assert.True(t, aClosed)
	})

	t.Run("concurrent with Close", func(t *testing.T) {
		const concurrency = 10

		c, err := di.NewContainer()
		require.NoError(t, err)

		results := make([]error, concurrency)
		testutils.RunParallel(concurrency, func(i int) {
			results[i] = c.Close(context.Background())
		})

		numErrors := 0
		for _, err := range results {
			if err != nil {
				assert.EqualError(t, err, "di.Container.Close: closed already: container closed")
				numErrors++
			}
		}

		assert.Equal(t, concurrency-1, numErrors, "only one call should return a nil error")
	})

	t.Run("concurrent with Resolve", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}),
		)
		require.NoError(t, err)

		var closeErr, resolveErr error
		wg := sync.WaitGroup{}
		wg.Add(2)

		go func() {
			closeErr = c.Close(context.Background())
			wg.Done()
		}()

		go func() {
			_, resolveErr = di.Resolve[*testtypes.StructA](context.Background(), c)
			wg.Done()
		}()

		wg.Wait()

		assert.NoError(t, closeErr)
		if resolveErr != nil {
			assert.EqualError(t, resolveErr, "di.Container.Resolve *testtypes.StructA: container closed")
		}
	})

	t.Run("concurrent with NewScope", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		var closeErr, scopeErr error
		wg := sync.WaitGroup{}
		wg.Add(2)

		go func() {
			closeErr = c.Close(context.Background())
			wg.Done()
		}()

		go func() {
			_, scopeErr = c.NewScope()
			wg.Done()
		}()

		wg.Wait()

		assert.NoError(t, closeErr)
		if scopeErr != nil {
			assert.EqualError(t, scopeErr, "di.Container.NewScope: container closed")
		}
	})
}
