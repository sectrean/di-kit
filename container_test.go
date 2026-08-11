package di_test

import (
	"context"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/internal/errors"
	"github.com/sectrean/di-kit/internal/mocks"
	"github.com/sectrean/di-kit/internal/testtypes"
	. "github.com/sectrean/di-kit/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewContainer(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		c, err := di.NewContainer()
		assert.NotNil(t, c)
		assert.NoError(t, err)
	})

	t.Run("di.WithService", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		assert.NotNil(t, c)
		assert.NoError(t, err)

		assert.True(t, di.Contains[testtypes.InterfaceA](c))
	})

	t.Run("di.WithService invalid type int", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(1234),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService int: invalid service type")
	})

	t.Run("di.WithService interface nil", func(t *testing.T) {
		var a testtypes.InterfaceA = nil
		c, err := di.NewContainer(
			di.WithService(a),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService: funcOrValue is nil")
	})

	t.Run("di.WithService pointer nil", func(t *testing.T) {
		var a *testtypes.StructA = nil
		c, err := di.NewContainer(
			di.WithService(a),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService: funcOrValue is nil")
	})

	t.Run("di.WithService invalid type di.Lifetime", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(di.Singleton, di.WithTag("tag")),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService di.Lifetime: invalid service type")
	})

	t.Run("di.WithService invalid type map", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(map[string]int{}),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService map[string]int: invalid service type")
	})

	t.Run("di.WithService invalid type *int", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() *int { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func() *int: invalid service type")
	})

	t.Run("di.WithService func returns unnamed func", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() func(http.Handler) http.Handler { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func() func(http.Handler) http.Handler: invalid service type")
	})

	t.Run("di.WithService invalid dependency type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(int) testtypes.InterfaceA { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func(int) testtypes.InterfaceA: invalid dependency type int")
	})

	t.Run("di.WithService invalid dependency type error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(error) testtypes.InterfaceA { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func(error) testtypes.InterfaceA: invalid dependency type error")
	})

	t.Run("di.WithService invalid dependency types", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(int, di.Lifetime) testtypes.InterfaceA { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func(int, di.Lifetime) testtypes.InterfaceA: invalid dependency type int\n"+
			"invalid dependency type di.Lifetime")
	})

	t.Run("di.WithService di.As not assignable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.As[*testtypes.StructA]()),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func() testtypes.InterfaceA: di.As[*testtypes.StructA]: type testtypes.InterfaceA not assignable to *testtypes.StructA")
	})

	t.Run("di.WithService di.As invalid service type map", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.CustomMap{}, di.As[map[string]any]()),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService testtypes.CustomMap: di.As[map[string]interface {}]: invalid service type")
	})

	t.Run("di.WithService di.Singleton value service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}, di.Singleton),
		)
		assert.NotNil(t, c)
		assert.NoError(t, err)
	})

	t.Run("di.WithService di.Transient value service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}, di.Transient),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService *testtypes.StructA: di.Transient: invalid lifetime for value service")
	})

	t.Run("di.WithService di.Lifetime invalid", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}, di.Lifetime(99)),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService *testtypes.StructA: di.Lifetime(99): invalid lifetime")
	})

	t.Run("di.WithService di.As interface not assignable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}, di.As[testtypes.InterfaceB]()),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService *testtypes.StructA: di.As[testtypes.InterfaceB]: type *testtypes.StructA not assignable to testtypes.InterfaceB")
	})

	t.Run("di.WithService di.WithTagged parameter not found", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				di.WithTagged[testtypes.InterfaceB]("tag"),
			),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func() testtypes.InterfaceA: di.WithTagged[testtypes.InterfaceB]: parameter not found")
	})

	t.Run("di.WithService di.WithTag invalid tag", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithTag([]string{"tag"})),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func() testtypes.InterfaceA: "+
			"di.WithTag: invalid tag type []string: type must be comparable")
	})

	t.Run("di.WithService di.WithTag duplicate tag", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceA,
				di.WithTag(nil),
				di.WithTag("other"),
				di.WithTag("other"),
			),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func() testtypes.InterfaceA: "+
			"di.WithTag other: duplicate tag")
	})

	t.Run("di.WithService di.WithTagged invalid tag", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(
				func(testtypes.InterfaceA) testtypes.InterfaceB { return &testtypes.StructB{} },
				di.WithTagged[testtypes.InterfaceA]([]string{"tag"}),
			),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func(testtypes.InterfaceA) testtypes.InterfaceB: "+
			"di.WithTagged[testtypes.InterfaceA]: invalid tag type []string: type must be comparable")
	})

	t.Run("di.WithService di.UseCloseFunc not assignable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				di.UseCloseFunc(func(context.Context, *testtypes.StructA) error { return nil }),
			),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func() testtypes.InterfaceA: di.UseCloseFunc: service type testtypes.InterfaceA is not assignable to *testtypes.StructA")
	})

	t.Run("di.WithService unsupported func signature", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() (testtypes.InterfaceA, testtypes.InterfaceB) { return nil, nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err,
			"di.NewContainer: di.WithService func() (testtypes.InterfaceA, testtypes.InterfaceB): function must return Service or (Service, error)")
	})

	t.Run("di.WithService invalid type error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() error { return errors.New("test error") }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func() error: invalid service type")
	})

	t.Run("di.WithService invalid type context.Context", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(context.Background),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService func() context.Context: invalid service type")
	})

	t.Run("di.WithService invalid basic types", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService([]int{}),
			di.WithService(map[string]int{}),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService []int: invalid service type\n"+
			"di.WithService map[string]int: invalid service type",
		)
	})

	t.Run("multiple errors", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService([]testtypes.InterfaceA{}),
			di.WithService(testtypes.NewInterfaceA, di.As[testtypes.InterfaceB]()),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService []testtypes.InterfaceA: invalid service type\n"+
			"di.WithService func() testtypes.InterfaceA: di.As[testtypes.InterfaceB]: type testtypes.InterfaceA not assignable to testtypes.InterfaceB",
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
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.WithService []testtypes.InterfaceA: invalid service type\n"+
			"di.WithService func() testtypes.InterfaceA: di.As[testtypes.InterfaceB]: type testtypes.InterfaceA not assignable to testtypes.InterfaceB\n"+
			"di.WithTagged[*testtypes.StructB]: parameter not found",
		)
	})

	t.Run("di.Module", func(t *testing.T) {
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

	t.Run("di.Module di.WithService nil", func(t *testing.T) {
		c, err := di.NewContainer(
			di.Module{
				di.WithService(testtypes.NewInterfaceA),
				di.WithService(nil),
			},
			di.WithService(testtypes.NewInterfaceC),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: di.Module: di.WithService: funcOrValue is nil")
	})
}

func Test_Container_NewScope(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.Scoped),
		)
		require.NoError(t, err)

		scope, err := c.NewScope()
		assert.NotNil(t, scope)
		assert.NoError(t, err)

		assert.True(t, di.Contains[testtypes.InterfaceA](c))
		assert.True(t, di.Contains[testtypes.InterfaceB](c))
	})

	t.Run("di.WithService", func(t *testing.T) {
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

	t.Run("di.WithService invalid type di.Lifetime", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(di.Scoped),
		)
		LogError(t, err)

		assert.Nil(t, scope)
		assert.EqualError(t, err, "di.Container.NewScope: di.WithService di.Lifetime: invalid service type")
		assert.NoError(t, c.Close(context.Background()), "a failed NewScope must end use of its parent")
	})

	t.Run("parent closed", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		err = c.Close(ctx)
		assert.NoError(t, err)

		scope, err := c.NewScope()
		LogError(t, err)

		assert.Nil(t, scope)
		assert.EqualError(t, err, "di.Container.NewScope: container closed")
	})

	t.Run("scoped service registered with tag", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(testtypes.NewInterfaceB, di.WithTag("tag")),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceB](ctx, scope, di.WithTag("tag"))
		assert.NotNil(t, got)
		assert.NoError(t, err)

		// The tag must match exactly: the untagged key is not registered.
		assert.False(t, di.Contains[testtypes.InterfaceB](scope))
	})

	t.Run("scoped service registered with As", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		a := &testtypes.StructA{}
		scope, err := c.NewScope(
			di.WithService(a,
				di.As[testtypes.InterfaceA](),
				di.As[*testtypes.StructA](),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()

		gotIface, err := di.Resolve[testtypes.InterfaceA](ctx, scope)
		assert.NoError(t, err)
		assert.Same(t, a, gotIface)

		gotPtr, err := di.Resolve[*testtypes.StructA](ctx, scope)
		assert.NoError(t, err)
		assert.Same(t, a, gotPtr)
	})

	t.Run("last scoped service registered wins", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		first := &testtypes.StructA{Tag: "first"}
		last := &testtypes.StructA{Tag: "last"}

		scope, err := c.NewScope(
			di.WithService(first),
			di.WithService(last),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[*testtypes.StructA](ctx, scope)
		assert.NoError(t, err)
		assert.Same(t, last, got)
	})
}

func Test_Container_Contains(t *testing.T) {
	t.Run("service registered", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		has := di.Contains[testtypes.InterfaceA](c)
		assert.True(t, has)
	})

	t.Run("service not registered", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		has := di.Contains[testtypes.InterfaceA](c)
		assert.False(t, has)
	})

	t.Run("di.WithTag", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithTag("tag")),
		)
		require.NoError(t, err)

		has := di.Contains[testtypes.InterfaceA](c, di.WithTag("tag"))
		assert.True(t, has)

		has = di.Contains[testtypes.InterfaceA](c)
		assert.False(t, has)

		has = c.Contains(reflect.TypeFor[testtypes.InterfaceA](), di.WithTag("other"))
		assert.False(t, has)
	})

	t.Run("di.WithTag invalid tag", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		var has bool
		require.NotPanics(t, func() {
			has = di.Contains[testtypes.InterfaceA](c, di.WithTag([]string{"tag"}))
		})
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

		has := di.Contains[testtypes.InterfaceA](scope)
		assert.True(t, has)

		has = di.Contains[testtypes.InterfaceB](scope)
		assert.True(t, has)
	})

	t.Run("slice service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		has := di.Contains[[]testtypes.InterfaceA](c)
		assert.True(t, has)
	})

	t.Run("slice service not registered", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		has := di.Contains[[]testtypes.InterfaceB](c)
		assert.False(t, has)
	})

	t.Run("di.WithTag slice service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithTag(1)),
		)
		require.NoError(t, err)

		has := di.Contains[[]testtypes.InterfaceA](c)
		assert.False(t, has)

		has = di.Contains[[]testtypes.InterfaceA](c, di.WithTag(1))
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

	t.Run("di.As func pointer nil", func(t *testing.T) {
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
		LogError(t, err)

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
		LogError(t, err)

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
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: context deadline exceeded")
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("not registered service", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: service not registered")
	})

	t.Run("not registered di.Scope", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[di.Scope](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve di.Scope: service not registered")
	})

	t.Run("not registered context.Context", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[context.Context](ctx, c)
		LogError(t, err)

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
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: dependency testtypes.InterfaceA: service not registered")
	})

	t.Run("dependency cycle", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceA { return nil }),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceB { return nil }),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: dependency testtypes.InterfaceB: dependency testtypes.InterfaceA: dependency cycle detected")
	})

	t.Run("dependency cycle transient", func(t *testing.T) {
		// A transient service that depends on itself is detected on a different
		// code path than singleton/scoped cycles: transient results are not cached,
		// so the cycle is caught by the resolve visitor rather than an in-flight result.
		c, err := di.NewContainer(
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceA {
				return &testtypes.StructA{}
			}, di.Transient),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: dependency testtypes.InterfaceA: dependency cycle detected")
	})

	t.Run("dependency cycle transient mutual", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceA {
				t.Error("constructor should not get called")
				return nil
			}, di.Transient),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceB {
				t.Error("constructor should not get called")
				return nil
			}, di.Transient),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: dependency testtypes.InterfaceB: dependency testtypes.InterfaceA: dependency cycle detected")
	})

	t.Run("di.Singleton", func(t *testing.T) {
		calls := 0

		c, err := di.NewContainer(
			di.WithService(
				func() testtypes.InterfaceA {
					calls++
					return &testtypes.StructA{Tag: 1}
				},
				di.Singleton,
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

	t.Run("di.Singleton from child scope", func(t *testing.T) {
		calls := 0

		c, err := di.NewContainer(
			di.WithService(
				func() testtypes.InterfaceA {
					calls++
					return &testtypes.StructA{Tag: 1}
				},
				di.Singleton,
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

	t.Run("di.Transient", func(t *testing.T) {
		calls := 0

		c, err := di.NewContainer(
			di.WithService(
				func() testtypes.InterfaceA {
					calls++
					return &testtypes.StructA{Tag: calls}
				},
				di.Transient,
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

	t.Run("di.Scoped", func(t *testing.T) {
		calls := 0

		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(
				func(a testtypes.InterfaceA) testtypes.InterfaceB {
					calls++
					assert.NotNil(t, a)
					return &testtypes.StructB{}
				},
				di.Scoped,
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

	t.Run("di.Scoped resolve from root", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.Scoped),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: scoped service must be resolved from a child scope")
	})

	t.Run("di.Scoped multi level", func(t *testing.T) {
		root, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		scope1, err := root.NewScope(
			di.WithService(testtypes.NewInterfaceB, di.Scoped),
		)
		require.NoError(t, err)

		ctx := context.Background()
		b, err := di.Resolve[testtypes.InterfaceB](ctx, scope1)
		LogError(t, err)

		assert.Nil(t, b)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: scoped service must be resolved from a child scope")

		scope2, err := scope1.NewScope()
		require.NoError(t, err)

		b, err = di.Resolve[testtypes.InterfaceB](ctx, scope2)
		assert.NotNil(t, b)
		assert.NoError(t, err)
	})

	t.Run("di.Scoped dependencies", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.Singleton),
			di.WithService(testtypes.NewInterfaceC, di.Scoped),
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

	t.Run("di.Scoped captive dependency", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.Scoped),
			di.WithService(testtypes.NewInterfaceB, di.Singleton),
		)
		require.NoError(t, err)

		ctx := context.Background()
		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		LogError(t, err)

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
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve []testtypes.InterfaceA: service func() (testtypes.InterfaceA, error): test error")
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

		want := []*testtypes.StructA{nil, a1}
		assert.Equal(t, want, got, "nil service should be included in the slice")
		assert.NoError(t, err)
	})

	t.Run("slice service not registered", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[[]testtypes.InterfaceA](ctx, c)
		LogError(t, err)

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
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve []testtypes.InterfaceA {Tag 1}: service not registered")
	})

	t.Run("WithTag multiple tags", func(t *testing.T) {
		a1 := &testtypes.StructA{Tag: 1}
		a2 := &testtypes.StructA{Tag: 2}

		c, err := di.NewContainer(
			di.WithService(a1,
				di.As[testtypes.InterfaceA](),
				di.WithTag(nil),
				di.WithTag("a"),
				di.WithTag("b"),
			),
			di.WithService(a2,
				di.As[testtypes.InterfaceA](),
				di.WithTag("a"),
				di.WithTag("b"),
				di.WithTag("c"),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()

		got := di.MustResolve[testtypes.InterfaceA](ctx, c)
		assert.Same(t, a1, got)

		gotA := di.MustResolve[testtypes.InterfaceA](ctx, c, di.WithTag("a"))
		assert.Same(t, a2, gotA)

		gotSliceA := di.MustResolve[[]testtypes.InterfaceA](ctx, c, di.WithTag("a"))
		assert.Equal(t, []testtypes.InterfaceA{a1, a2}, gotSliceA)

		gotSliceB := di.MustResolve[[]testtypes.InterfaceA](ctx, c, di.WithTag("b"))
		assert.Equal(t, []testtypes.InterfaceA{a1, a2}, gotSliceB)
	})

	t.Run("di.As", func(t *testing.T) {
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
		LogError(t, err)

		assert.Nil(t, got)
	})

	t.Run("di.As original type", func(t *testing.T) {
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
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA {Tag other}: service not registered")
	})

	t.Run("WithTag invalid tag", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		var got testtypes.InterfaceA
		var resolveErr error
		require.NotPanics(t, func() {
			got, resolveErr = di.Resolve[testtypes.InterfaceA](ctx, c, di.WithTag([]string{"tag"}))
		})

		assert.Nil(t, got)
		assert.EqualError(t, resolveErr, "di.Container.Resolve testtypes.InterfaceA: "+
			"di.WithTag: invalid tag type []string: type must be comparable")
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
		LogError(t, err)

		assert.Nil(t, a)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: constructor error")

		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		LogError(t, err)

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
		ctx := ContextWithTestValue(context.Background(), "value")

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
			di.WithService(func(ctx context.Context, scope di.Scope) *testtypes.ScopeFactory[testtypes.InterfaceA] {
				// We cannot call Resolve on the scope here.
				a, err := di.Resolve[testtypes.InterfaceA](ctx, scope)
				LogError(t, err)

				assert.Nil(t, a)
				assert.EqualError(t, err,
					"di.Scope.Resolve testtypes.InterfaceA: "+
						"resolve not allowed within service constructor function")

				// Contains can be called though
				assert.True(t, di.Contains[testtypes.InterfaceA](scope))

				// We have to store it and we can call Resolve later.
				return testtypes.NewScopeFactory(scope, func(ctx context.Context, s di.Scope) (testtypes.InterfaceA, error) {
					return di.Resolve[testtypes.InterfaceA](ctx, s)
				})
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		factory, err := di.Resolve[*testtypes.ScopeFactory[testtypes.InterfaceA]](ctx, c)
		require.NoError(t, err)

		a, err := factory.Build(ctx)
		assert.NotNil(t, a)
		assert.NoError(t, err)
	})

	t.Run("di.Module", func(t *testing.T) {
		// The module service should be registered first since the module is added before the
		// other service registrations.
		a1 := &testtypes.StructA{Tag: 1}
		a2 := &testtypes.StructA{Tag: 2}

		c, err := di.NewContainer(
			di.Module{
				di.WithService(a1, di.As[testtypes.InterfaceA]()),
				di.WithService(testtypes.NewInterfaceB),
			},
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

		Go(&wg, func() {
			a, err := di.Resolve[testtypes.InterfaceA](ctx, c)
			assert.NotNil(t, a)
			assert.NoError(t, err)
		})
		Go(&wg, func() {
			b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
			assert.NotNil(t, b)
			assert.NoError(t, err)
		})
		Go(&wg, func() {
			c, err := di.Resolve[testtypes.InterfaceC](ctx, c)
			assert.NotNil(t, c)
			assert.NoError(t, err)
		})
		Go(&wg, func() {
			d, err := di.Resolve[testtypes.InterfaceD](ctx, c)
			assert.NotNil(t, d)
			assert.NoError(t, err)
		})

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
			Go(&wg, func() {
				got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
				assert.Same(t, expected, got)
				assert.NoError(t, err)
			})
		}

		wg.Wait()
		assert.Equal(t, 1, calls)
	})

	t.Run("concurrent scoped", func(t *testing.T) {
		const n = 100
		starts := atomic.Int32{}
		allStarted := make(chan struct{})
		calls := 0

		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.Transient),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceB {
				// Only one goroutine constructs the scoped service; the count is
				// written without synchronization so -race flags a second construction.
				calls++
				// Block until every goroutine has begun resolving, forcing the others
				// to wait on this in-flight result instead of finding it already done.
				<-allStarted
				return &testtypes.StructB{}
			}, di.Scoped),
		)
		require.NoError(t, err)

		scope, err := c.NewScope()
		require.NoError(t, err)

		ctx := context.Background()
		wg := sync.WaitGroup{}

		for range n {
			Go(&wg, func() {
				if starts.Add(1) == n {
					close(allStarted)
				}
				got, err := di.Resolve[testtypes.InterfaceB](ctx, scope)
				assert.NotNil(t, got)
				assert.NoError(t, err)
			})
		}

		wg.Wait()
		assert.Equal(t, 1, calls)
	})

	t.Run("concurrent singleton race", func(t *testing.T) {
		const n = 2
		starts := atomic.Int32{}
		allStarted := make(chan struct{})
		calls := 0

		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceB {
				// Only one goroutine constructs the singleton; the count is written
				// without synchronization so -race flags a second construction.
				calls++
				// Block until both goroutines are resolving, so the second waits on
				// this in-flight result instead of racing to construct its own.
				<-allStarted
				return &testtypes.StructB{}
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		wg := sync.WaitGroup{}

		for range n {
			Go(&wg, func() {
				if starts.Add(1) == n {
					close(allStarted)
				}
				_, err := di.Resolve[testtypes.InterfaceB](ctx, c)
				assert.NoError(t, err)
			})
		}

		wg.Wait()
		assert.Equal(t, 1, calls)
	})

	t.Run("concurrent singletons", func(t *testing.T) {
		const barrierTimeout = 2 * time.Second

		var inConstruction atomic.Int32
		var maxObserved atomic.Int32

		// waitForBoth blocks until both constructors have been observed in-flight
		// at the same time, updating maxObserved.
		// It waits on the monotonic high-water mark rather than the live counter:
		// the other constructor returns as soon as it observes the overlap, so the
		// live counter can drop back down between polls.
		waitForBoth := func() {
			deadline := time.After(barrierTimeout)
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()
			for {
				cur := inConstruction.Load()
				// Record the high-water mark.
				for {
					m := maxObserved.Load()
					if cur <= m || maxObserved.CompareAndSwap(m, cur) {
						break
					}
				}
				if maxObserved.Load() >= 2 {
					return
				}
				select {
				case <-deadline:
					return
				case <-ticker.C:
				}
			}
		}

		newA := func() *testtypes.StructA {
			inConstruction.Add(1)
			defer inConstruction.Add(-1)
			waitForBoth()
			return &testtypes.StructA{}
		}
		newB := func() *testtypes.StructB {
			inConstruction.Add(1)
			defer inConstruction.Add(-1)
			waitForBoth()
			return &testtypes.StructB{}
		}

		c, err := di.NewContainer(
			di.WithService(newA),
			di.WithService(newB),
		)
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.Close(context.Background()) })

		ctx := context.Background()

		var wg sync.WaitGroup
		var errA, errB error
		Go(&wg, func() {
			_, errA = di.Resolve[*testtypes.StructA](ctx, c)
		})
		Go(&wg, func() {
			_, errB = di.Resolve[*testtypes.StructB](ctx, c)
		})
		wg.Wait()

		assert.NoError(t, errA)
		assert.NoError(t, errB)

		// The definitive assertion: at some point both constructors were in-flight
		// simultaneously. A serialized implementation caps this at 1.
		assert.Equalf(t, int32(2), maxObserved.Load(),
			"expected both constructors to run concurrently; "+
				"max simultaneous in-construction was %d (constructors were serialized)",
			maxObserved.Load())
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
		Go(&wg, func() {
			_, err := di.Resolve[testtypes.InterfaceA](ctx, c)
			LogError(t, err)
			assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: dependency testtypes.InterfaceB: dependency testtypes.InterfaceA: dependency cycle detected")
		})

		Go(&wg, func() {
			_, err := di.Resolve[testtypes.InterfaceB](ctx, c)
			LogError(t, err)
			assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: dependency testtypes.InterfaceA: dependency testtypes.InterfaceB: dependency cycle detected")
		})

		wg.Wait()
	})

	t.Run("concurrent dependency cycle three goroutines", func(t *testing.T) {
		// A three-service cycle resolved from three goroutines can deadlock with each
		// goroutine owning one in-flight service while waiting on the next.
		// The exact error message depends on where the wait cycle is detected.
		c, err := di.NewContainer(
			di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceA {
				assert.Fail(t, "constructor func should not get called")
				return nil
			}),
			di.WithService(func(testtypes.InterfaceC) testtypes.InterfaceB {
				assert.Fail(t, "constructor func should not get called")
				return nil
			}),
			di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceC {
				assert.Fail(t, "constructor func should not get called")
				return nil
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		wg := sync.WaitGroup{}

		Go(&wg, func() {
			_, err := di.Resolve[testtypes.InterfaceA](ctx, c)
			LogError(t, err)
			assert.ErrorContains(t, err, "dependency cycle detected")
		})
		Go(&wg, func() {
			_, err := di.Resolve[testtypes.InterfaceB](ctx, c)
			LogError(t, err)
			assert.ErrorContains(t, err, "dependency cycle detected")
		})
		Go(&wg, func() {
			_, err := di.Resolve[testtypes.InterfaceC](ctx, c)
			LogError(t, err)
			assert.ErrorContains(t, err, "dependency cycle detected")
		})

		wg.Wait()
	})

	t.Run("concurrent dependency cycle detected across goroutines", func(t *testing.T) {
		// Deterministically exercise the cross-goroutine deadlock detection in
		// beginWaiting: each goroutine publishes its own in-flight service and only
		// then races to resolve the other's, so the cycle is found by following the
		// chain of waiting goroutines rather than by a single goroutine's visitor.
		//
		// The gate services block each top-level resolution after its in-flight
		// result is published but before its cyclic dependency is resolved, until
		// both goroutines have arrived.
		started := make(chan struct{}, 8)
		release := make(chan struct{})

		gate := func() {
			started <- struct{}{}
			select {
			case <-release:
			case <-time.After(2 * time.Second):
			}
		}

		c, err := di.NewContainer(
			// Gate services are resolved first, before each cyclic dependency.
			di.WithService(func() testtypes.InterfaceC { gate(); return &testtypes.StructC{} }),
			di.WithService(func() testtypes.InterfaceD { gate(); return &testtypes.StructD{} }),
			di.WithService(func(testtypes.InterfaceC, testtypes.InterfaceB) testtypes.InterfaceA {
				assert.Fail(t, "constructor func should not get called")
				return nil
			}),
			di.WithService(func(testtypes.InterfaceD, testtypes.InterfaceA) testtypes.InterfaceB {
				assert.Fail(t, "constructor func should not get called")
				return nil
			}),
		)
		require.NoError(t, err)

		// Release both goroutines once both in-flight results have been published.
		go func() {
			<-started
			<-started
			close(release)
		}()

		ctx := context.Background()
		wg := sync.WaitGroup{}
		Go(&wg, func() {
			_, err := di.Resolve[testtypes.InterfaceA](ctx, c)
			LogError(t, err)
			assert.ErrorContains(t, err, "dependency cycle detected")
		})
		Go(&wg, func() {
			_, err := di.Resolve[testtypes.InterfaceB](ctx, c)
			LogError(t, err)
			assert.ErrorContains(t, err, "dependency cycle detected")
		})

		wg.Wait()
	})

	t.Run("concurrent dependency cycle detected across goroutines in child scope", func(t *testing.T) {
		// Like the test above, but the cyclic services are Scoped and resolved from
		// a child scope, so the cross-goroutine deadlock detection runs against the
		// root container reached by walking up from the child scope.
		started := make(chan struct{}, 8)
		release := make(chan struct{})

		gate := func() {
			started <- struct{}{}
			select {
			case <-release:
			case <-time.After(2 * time.Second):
			}
		}

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceC { gate(); return &testtypes.StructC{} }, di.Scoped),
			di.WithService(func() testtypes.InterfaceD { gate(); return &testtypes.StructD{} }, di.Scoped),
			di.WithService(func(testtypes.InterfaceC, testtypes.InterfaceB) testtypes.InterfaceA {
				assert.Fail(t, "constructor func should not get called")
				return nil
			}, di.Scoped),
			di.WithService(func(testtypes.InterfaceD, testtypes.InterfaceA) testtypes.InterfaceB {
				assert.Fail(t, "constructor func should not get called")
				return nil
			}, di.Scoped),
		)
		require.NoError(t, err)

		scope, err := c.NewScope()
		require.NoError(t, err)

		go func() {
			<-started
			<-started
			close(release)
		}()

		ctx := context.Background()
		wg := sync.WaitGroup{}
		Go(&wg, func() {
			_, err := di.Resolve[testtypes.InterfaceA](ctx, scope)
			LogError(t, err)
			assert.ErrorContains(t, err, "dependency cycle detected")
		})
		Go(&wg, func() {
			_, err := di.Resolve[testtypes.InterfaceB](ctx, scope)
			LogError(t, err)
			assert.ErrorContains(t, err, "dependency cycle detected")
		})

		wg.Wait()
	})

	t.Run("concurrent singleton retry after cancel", func(t *testing.T) {
		started := make(chan struct{})
		calls := atomic.Int32{}

		c, err := di.NewContainer(
			di.WithService(func(ctx context.Context) (testtypes.InterfaceA, error) {
				if calls.Add(1) == 1 {
					// The first call blocks until its context is canceled.
					close(started)
					<-ctx.Done()
					return nil, ctx.Err()
				}
				return &testtypes.StructA{}, nil
			}),
		)
		require.NoError(t, err)

		ctx1, cancel := context.WithCancel(context.Background())
		var err1 error

		wg := sync.WaitGroup{}
		Go(&wg, func() {
			_, err1 = di.Resolve[testtypes.InterfaceA](ctx1, c)
		})

		// Wait until the first goroutine is in the constructor function,
		// then resolve with a context that is not canceled.
		<-started

		var got2 testtypes.InterfaceA
		var err2 error
		Go(&wg, func() {
			got2, err2 = di.Resolve[testtypes.InterfaceA](context.Background(), c)
		})

		// Give the second goroutine a chance to start waiting on the in-flight result
		// before failing the first resolve.
		time.Sleep(10 * time.Millisecond)
		cancel()
		wg.Wait()

		// The canceled error must not be cached:
		// the second resolve retries with a new constructor call and succeeds.
		assert.ErrorIs(t, err1, context.Canceled)
		assert.NoError(t, err2)
		assert.NotNil(t, got2)
		assert.Equal(t, int32(2), calls.Load())
	})

	t.Run("concurrent resolve canceled while waiting", func(t *testing.T) {
		// One goroutine blocks in the constructor while a second waits on the
		// in-flight result. Canceling the waiting goroutine's context must return
		// it promptly with the cancellation error, without affecting the constructor.
		constructing := make(chan struct{})
		release := make(chan struct{})

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				close(constructing)
				<-release
				return &testtypes.StructA{}
			}),
		)
		require.NoError(t, err)

		// The first goroutine constructs (blocking until released) with an
		// uncancelable context, publishing the in-flight result.
		g1 := make(chan error, 1)
		go func() {
			_, err := di.Resolve[testtypes.InterfaceA](context.Background(), c)
			g1 <- err
		}()
		<-constructing

		// The second goroutine waits on that in-flight result with a cancelable context.
		ctx2, cancel := context.WithCancel(context.Background())
		g2 := make(chan error, 1)
		go func() {
			_, err := di.Resolve[testtypes.InterfaceA](ctx2, c)
			g2 <- err
		}()

		// Give the second goroutine a chance to start waiting, then cancel it.
		// The constructor stays blocked, so this deterministically exercises the
		// context-canceled branch of the wait rather than returning a cached result.
		time.Sleep(10 * time.Millisecond)
		cancel()

		select {
		case err2 := <-g2:
			assert.ErrorIs(t, err2, context.Canceled)
		case <-time.After(2 * time.Second):
			assert.Fail(t, "canceled resolve did not return while waiting")
		}

		// Releasing the constructor still lets the first resolve succeed.
		close(release)
		assert.NoError(t, <-g1)
	})

	t.Run("concurrent singleton constructor panic", func(t *testing.T) {
		started := make(chan struct{})
		calls := atomic.Int32{}

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				if calls.Add(1) == 1 {
					// The first call panics after giving the second goroutine
					// a chance to start waiting on the in-flight result.
					close(started)
					time.Sleep(10 * time.Millisecond)
					panic("constructor panic")
				}
				return &testtypes.StructA{}
			}),
		)
		require.NoError(t, err)

		wg := sync.WaitGroup{}
		Go(&wg, func() {
			defer func() {
				// The panic propagates to the goroutine that called the constructor function
				assert.Equal(t, "constructor panic", recover())
			}()

			_, _ = di.Resolve[testtypes.InterfaceA](context.Background(), c)
			assert.Fail(t, "expected constructor func to panic")
		})

		// Wait until the first goroutine is in the constructor function
		<-started

		var got testtypes.InterfaceA
		var resolveErr error
		Go(&wg, func() {
			got, resolveErr = di.Resolve[testtypes.InterfaceA](context.Background(), c)
		})

		wg.Wait()

		// The second goroutine must not wait forever on the panicked resolve:
		// it retries with a new constructor call and succeeds.
		assert.NoError(t, resolveErr)
		assert.NotNil(t, got)
		assert.Equal(t, int32(2), calls.Load())
	})

	t.Run("concurrent scopes stress", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceAStruct),
			di.WithService(testtypes.NewInterfaceB, di.Transient),
			di.WithService(testtypes.NewInterfaceC, di.Scoped),
		)
		require.NoError(t, err)

		ctx := context.Background()

		RunParallel(10, func(int) {
			for range 10 {
				scope, scopeErr := c.NewScope()
				if !assert.NoError(t, scopeErr) {
					return
				}

				gotC, resolveErr := di.Resolve[testtypes.InterfaceC](ctx, scope)
				assert.NotNil(t, gotC)
				assert.NoError(t, resolveErr)

				gotAs, resolveErr := di.Resolve[[]testtypes.InterfaceA](ctx, scope)
				assert.Len(t, gotAs, 2)
				assert.NoError(t, resolveErr)

				closeErr := scope.Close(ctx)
				assert.NoError(t, closeErr)
			}
		})

		err = c.Close(ctx)
		assert.NoError(t, err)
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

		Go(&wg, func() {
			b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
			assert.NotNil(t, b)
			assert.NoError(t, err)
		})

		Go(&wg, func() {
			c, err := di.Resolve[testtypes.InterfaceC](ctx, c)
			assert.NotNil(t, c)
			assert.NoError(t, err)

			c.C()
		})

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
		LogError(t, err)

		assert.EqualError(t, err, "di.Container.Close: container already closed")
	})

	t.Run("in use by child scope", func(t *testing.T) {
		value := &testtypes.StructA{Tag: "value"}
		c, err := di.NewContainer(
			di.WithService(value),
			di.WithService(func() testtypes.InterfaceB { return &testtypes.StructB{} }),
		)
		require.NoError(t, err)

		ctx := context.Background()
		cached, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		require.NoError(t, err)

		scope, err := c.NewScope()
		require.NoError(t, err)

		err = c.Close(ctx)
		assert.EqualError(t, err, "di.Container.Close: container in use")

		// A rejected Close must leave the parent usable. In particular, values and
		// cached singletons reached by walking through the child remain valid.
		gotValue, err := di.Resolve[*testtypes.StructA](ctx, scope)
		assert.NoError(t, err)
		assert.Same(t, value, gotValue)

		gotCached, err := di.Resolve[testtypes.InterfaceB](ctx, scope)
		assert.NoError(t, err)
		assert.Same(t, cached, gotCached)

		require.NoError(t, scope.Close(ctx))
		assert.NoError(t, c.Close(ctx))
	})

	t.Run("descendant scopes remain open", func(t *testing.T) {
		root, err := di.NewContainer()
		require.NoError(t, err)
		child, err := root.NewScope()
		require.NoError(t, err)
		grandchild, err := child.NewScope()
		require.NoError(t, err)

		ctx := context.Background()
		assert.EqualError(t, root.Close(ctx), "di.Container.Close: container in use")
		assert.EqualError(t, child.Close(ctx), "di.Container.Close: container in use")

		require.NoError(t, grandchild.Close(ctx))
		require.NoError(t, child.Close(ctx))
		assert.NoError(t, root.Close(ctx))
	})

	t.Run("child keeps parent in use while closing", func(t *testing.T) {
		closerStarted := make(chan struct{})
		releaseCloser := make(chan struct{})

		root, err := di.NewContainer()
		require.NoError(t, err)
		child, err := root.NewScope(
			di.WithService(&testtypes.StructA{},
				di.UseCloseFunc(func(context.Context, *testtypes.StructA) error {
					close(closerStarted)
					<-releaseCloser
					return nil
				}),
			),
		)
		require.NoError(t, err)

		closeDone := make(chan error, 1)
		go func() {
			closeDone <- child.Close(context.Background())
		}()

		<-closerStarted
		assert.EqualError(t, root.Close(context.Background()), "di.Container.Close: container in use")

		close(releaseCloser)
		require.NoError(t, <-closeDone)
		assert.NoError(t, root.Close(context.Background()))
	})

	t.Run("all close funcs", func(t *testing.T) {
		ctx := ContextWithTestValue(context.Background(), "value")

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
		LogError(t, err)
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
		LogError(t, err)
		assert.EqualError(t, err, "di.Container.Close: err c\nerr a")
	})

	t.Run("di.IgnoreCloser func service", func(t *testing.T) {
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

	t.Run("di.UseCloser value service", func(t *testing.T) {
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

	t.Run("di.UseCloseFunc func service", func(t *testing.T) {
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

	t.Run("di.UseCloseFunc value service", func(t *testing.T) {
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
		RunParallel(concurrency, func(i int) {
			results[i] = c.Close(context.Background())
		})

		numErrors := 0
		for _, err := range results {
			if err != nil {
				assert.Contains(t, []string{
					"di.Container.Close: container already closed",
					"di.Container.Close: container in use",
				}, err.Error())
				numErrors++
			}
		}

		assert.Equal(t, concurrency-1, numErrors, "only one call should return a nil error")
	})

	t.Run("resolution in progress", func(t *testing.T) {
		started := make(chan struct{})
		release := make(chan struct{})

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				close(started)
				<-release
				return &testtypes.StructA{}
			}),
		)
		require.NoError(t, err)

		var resolveErr error
		wg := sync.WaitGroup{}
		Go(&wg, func() {
			_, resolveErr = di.Resolve[testtypes.InterfaceA](context.Background(), c)
		})

		<-started
		closeErr := c.Close(context.Background())
		assert.EqualError(t, closeErr, "di.Container.Close: container in use")

		close(release)
		wg.Wait()
		assert.NoError(t, resolveErr)
		assert.NoError(t, c.Close(context.Background()))
	})

	t.Run("concurrent with Resolve on child scope", func(t *testing.T) {
		ctx := context.Background()

		started := make(chan struct{})
		release := make(chan struct{})

		aMock := mocks.NewInterfaceAMock(t)
		aMock.EXPECT().
			Close(ctx).
			Return(nil).
			Once()

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				close(started)
				<-release
				return aMock
			}),
		)
		require.NoError(t, err)

		scope, err := c.NewScope()
		require.NoError(t, err)

		var resolveErr error
		wg := sync.WaitGroup{}
		Go(&wg, func() {
			_, resolveErr = di.Resolve[testtypes.InterfaceA](ctx, scope)
		})

		// The open child prevents its parent from closing, including while the child
		// is resolving a singleton registered with that parent.
		<-started
		err = c.Close(ctx)
		assert.EqualError(t, err, "di.Container.Close: container in use")

		close(release)
		wg.Wait()

		assert.NoError(t, resolveErr)
		require.NoError(t, scope.Close(ctx))
		assert.NoError(t, c.Close(ctx))
	})

	t.Run("concurrent with Resolve on child scopes stress", func(t *testing.T) {
		const concurrency = 10

		var constructed, closed atomic.Int32

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				constructed.Add(1)
				time.Sleep(time.Millisecond)
				return &testtypes.StructA{}
			},
				di.UseCloseFunc(func(context.Context, testtypes.InterfaceA) error {
					closed.Add(1)
					return nil
				}),
			),
		)
		require.NoError(t, err)

		scopes := make([]*di.Container, concurrency)
		for i := range scopes {
			scope, err := c.NewScope()
			require.NoError(t, err)
			scopes[i] = scope
		}

		var closeErr error
		RunParallel(concurrency+1, func(i int) {
			if i == concurrency {
				closeErr = c.Close(context.Background())
				return
			}

			got, err := di.Resolve[testtypes.InterfaceA](context.Background(), scopes[i])
			assert.NoError(t, err)
			assert.NotNil(t, got)
		})

		assert.EqualError(t, closeErr, "di.Container.Close: container in use")
		for _, scope := range scopes {
			require.NoError(t, scope.Close(context.Background()))
		}
		require.NoError(t, c.Close(context.Background()))

		assert.Equal(t, int32(1), constructed.Load())
		assert.Equal(t, int32(1), closed.Load())
	})

	t.Run("concurrent with NewScope", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		var closeErr, scopeErr error
		var scope *di.Container
		wg := sync.WaitGroup{}

		Go(&wg, func() {
			closeErr = c.Close(context.Background())
		})
		Go(&wg, func() {
			scope, scopeErr = c.NewScope()
		})

		wg.Wait()

		if closeErr == nil {
			assert.EqualError(t, scopeErr, "di.Container.NewScope: container closed")
			assert.Nil(t, scope)
		} else {
			assert.EqualError(t, closeErr, "di.Container.Close: container in use")
			require.NoError(t, scopeErr)
			require.NoError(t, scope.Close(context.Background()))
			assert.NoError(t, c.Close(context.Background()))
		}
	})
}
