package di_test

import (
	"context"
	stderrors "errors"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/internal/errors"
	"github.com/sectrean/di-kit/internal/mocks"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Add tests for the following:
// - dependencies on scoped services
// - slices with scopes
// - slices with scoped services
// - decorators registered on child scopes
// - decorator on parent scope with scoped dependency
// - more tests around the resolve locking

func Test_NewContainer(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		c, err := di.NewContainer()
		assert.NotNil(t, c)
		assert.NoError(t, err)
	})

	t.Run("with service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		assert.NotNil(t, c)
		assert.NoError(t, err)

		has := c.Contains(reflect.TypeFor[testtypes.InterfaceA]())
		assert.True(t, has)
	})

	t.Run("with invalid service kind", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(1234),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService int: invalid service type")
	})

	t.Run("with nil value", func(t *testing.T) {
		var a testtypes.InterfaceA
		c, err := di.NewContainer(
			di.WithService(a),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService: funcOrValue is nil")
	})

	t.Run("invalid service type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(di.SingletonLifetime, di.WithTag("tag")),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService di.Lifetime: invalid service type")
	})

	t.Run("invalid service type pointer to invalid type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(map[string]int{}),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService map[string]int: invalid service type")
	})

	t.Run("invalid service type func returns pointer to invalid type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() *int { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() *int: invalid service type")
	})

	t.Run("invalid dependency type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(int) testtypes.InterfaceA { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func(int) testtypes.InterfaceA: invalid dependency type int")
	})

	t.Run("invalid dependency types", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(int, di.Lifetime) testtypes.InterfaceA { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func(int, di.Lifetime) testtypes.InterfaceA: invalid dependency type int\n"+
			"invalid dependency type di.Lifetime")
	})

	t.Run("func alias not assignable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.As[*testtypes.StructA]()),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() testtypes.InterfaceA: As *testtypes.StructA: type testtypes.InterfaceA not assignable to *testtypes.StructA")
	})

	t.Run("value alias not assignable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}, di.As[testtypes.InterfaceB]()),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService *testtypes.StructA: As testtypes.InterfaceB: type *testtypes.StructA not assignable to testtypes.InterfaceB")
	})

	t.Run("with tagged not found", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				di.WithTagged[testtypes.InterfaceB]("tag"),
			),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() testtypes.InterfaceA: WithTagged testtypes.InterfaceB: parameter not found")
	})

	t.Run("value with tagged", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				// This option will be ignored.
				di.WithTagged[testtypes.InterfaceB]("tag"),
			),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() testtypes.InterfaceA: WithTagged testtypes.InterfaceB: parameter not found")
	})

	t.Run("with close func not assignable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				di.WithCloseFunc(func(context.Context, *testtypes.StructA) error { return nil }),
			),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() testtypes.InterfaceA: WithCloseFunc: service type testtypes.InterfaceA is not assignable to *testtypes.StructA")
	})

	t.Run("unsupported func signature", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() (testtypes.InterfaceA, testtypes.InterfaceB) { return nil, nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err,
			"di.NewContainer: WithService func() (testtypes.InterfaceA, testtypes.InterfaceB): function must return Service or (Service, error)")
	})

	t.Run("register error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() error { return stderrors.New("test error") }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() error: invalid service type")
	})

	t.Run("register context.Context", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(context.Background),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService func() context.Context: invalid service type")
	})

	t.Run("multiple errors", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService([]testtypes.InterfaceA{}),
			di.WithService(testtypes.NewInterfaceA, di.As[testtypes.InterfaceB]()),
		)
		LogError(t, err)

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
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService []testtypes.InterfaceA: invalid service type\n"+
			"WithService func() testtypes.InterfaceA: As testtypes.InterfaceB: type testtypes.InterfaceA not assignable to testtypes.InterfaceB\n"+
			"WithTagged *testtypes.StructB: parameter not found",
		)
	})

	t.Run("with nil decorator", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithDecorator(nil),
		)
		LogError(t, err)
		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithDecorator: decorateFunc is nil")
	})

	t.Run("with decorator function with no service parameter", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithDecorator(func() testtypes.InterfaceA { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithDecorator func() testtypes.InterfaceA: function must have a Service parameter")
	})

	t.Run("with decorator invalid di.Lifetime", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithDecorator(di.SingletonLifetime),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithDecorator di.Lifetime: invalid decorator type")
	})

	t.Run("with decorator invalid di.As", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithDecorator(di.As[testtypes.InterfaceA]()),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithDecorator di.serviceOption: invalid decorator type")
	})

	t.Run("with decorator invalid func", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithDecorator(func(testtypes.InterfaceA) {}),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithDecorator func(testtypes.InterfaceA): function must return Service")
	})

	t.Run("with decorator invalid func return", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithDecorator(func(testtypes.InterfaceA) error { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithDecorator func(testtypes.InterfaceA) error: invalid service type")
	})

	t.Run("with decorator with tagged not found", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithDecorator(func(testtypes.InterfaceA) testtypes.InterfaceA {
				return nil
			}, di.WithTagged[testtypes.InterfaceB]("tag")),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithDecorator func(testtypes.InterfaceA) testtypes.InterfaceA: WithTagged testtypes.InterfaceB: parameter not found")
	})

	t.Run("with decorator invalid dependency type", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithDecorator(func(int, testtypes.InterfaceA) testtypes.InterfaceA { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithDecorator func(int, testtypes.InterfaceA) testtypes.InterfaceA: invalid dependency type int")
	})

	t.Run("with decorator invalid dependency types", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithDecorator(func(int, di.Lifetime, testtypes.InterfaceA) testtypes.InterfaceA { return nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithDecorator func(int, di.Lifetime, testtypes.InterfaceA) testtypes.InterfaceA: invalid dependency type int\n"+
			"invalid dependency type di.Lifetime")
	})

	t.Run("with module", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithModule(di.Module{
				di.WithService(testtypes.NewInterfaceA),
				di.WithService(testtypes.NewInterfaceB),
			}),
			di.WithService(testtypes.NewInterfaceC),
		)
		assert.NotNil(t, c)
		assert.NoError(t, err)
	})

	t.Run("with module error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithModule([]di.ContainerOption{
				di.WithService(testtypes.NewInterfaceA),
				di.WithService(nil),
			}),
			di.WithService(testtypes.NewInterfaceC),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "di.NewContainer: WithService: funcOrValue is nil")
	})
}

func Test_Container_NewScope(t *testing.T) {
	t.Run("no new services", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.ScopedLifetime),
		)
		require.NoError(t, err)

		scope, err := c.NewScope()
		assert.NotNil(t, scope)
		assert.NoError(t, err)

		assert.True(t, scope.Contains(reflect.TypeFor[testtypes.InterfaceA]()))
		assert.True(t, scope.Contains(reflect.TypeFor[testtypes.InterfaceB]()))
	})

	t.Run("with new service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(testtypes.NewInterfaceB),
		)
		assert.NotNil(t, scope)
		assert.NoError(t, err)

		assert.True(t, c.Contains(reflect.TypeFor[testtypes.InterfaceA]()))
		assert.False(t, c.Contains(reflect.TypeFor[testtypes.InterfaceB]()))

		assert.True(t, scope.Contains(reflect.TypeFor[testtypes.InterfaceA]()))
		assert.True(t, scope.Contains(reflect.TypeFor[testtypes.InterfaceB]()))
	})

	t.Run("with service error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(di.ScopedLifetime),
		)
		LogError(t, err)

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
		LogError(t, err)

		assert.Nil(t, scope)
		assert.EqualError(t, err, "di.Container.NewScope: container closed")
		assert.ErrorIs(t, err, di.ErrContainerClosed)
	})
}

func Test_Container_Contains(t *testing.T) {
	t.Run("type registered", func(t *testing.T) {
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

	t.Run("with tag", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithTag("tag")),
		)
		require.NoError(t, err)

		has := c.Contains(reflect.TypeFor[testtypes.InterfaceA](), di.WithTag("tag"))
		assert.True(t, has)

		has = c.Contains(reflect.TypeFor[testtypes.InterfaceA]())
		assert.True(t, has)

		has = c.Contains(reflect.TypeFor[testtypes.InterfaceA](), di.WithTag("other"))
		assert.False(t, has)
	})

	t.Run("child scope", func(t *testing.T) {
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

	t.Run("constructor func returns nil", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				return nil
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Nil(t, got)
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
		assert.ErrorIs(t, err, di.ErrServiceNotRegistered)
	})

	t.Run("not registered di.Scope", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[di.Scope](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve di.Scope: service not registered")
		assert.ErrorIs(t, err, di.ErrServiceNotRegistered)
	})

	t.Run("not registered context.Context", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[context.Context](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve context.Context: service not registered")
		assert.ErrorIs(t, err, di.ErrServiceNotRegistered)
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
		assert.ErrorIs(t, err, di.ErrServiceNotRegistered)
	})

	t.Run("dependency cycle", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceA { return nil }),
			di.WithService(testtypes.NewInterfaceB),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: dependency testtypes.InterfaceB: dependency testtypes.InterfaceA: dependency cycle detected")
		assert.ErrorIs(t, err, di.ErrDependencyCycle)
	})

	t.Run("lifetime singleton", func(t *testing.T) {
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

	t.Run("lifetime singleton from child scope", func(t *testing.T) {
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

	t.Run("lifetime transient", func(t *testing.T) {
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

	t.Run("lifetime scoped", func(t *testing.T) {
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

	t.Run("lifetime scoped resolve from root", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.ScopedLifetime),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: scoped service must be resolved from a child scope")
	})

	t.Run("lifetime scoped multi level", func(t *testing.T) {
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
		LogError(t, err)

		assert.Nil(t, b)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: scoped service must be resolved from a child scope")

		scope2, err := scope1.NewScope()
		require.NoError(t, err)

		b, err = di.Resolve[testtypes.InterfaceB](ctx, scope2)
		assert.NotNil(t, b)
		assert.NoError(t, err)
	})

	t.Run("lifetime scoped captive dependency", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.ScopedLifetime),
			di.WithService(testtypes.NewInterfaceB, di.SingletonLifetime),
		)
		require.NoError(t, err)

		ctx := context.Background()
		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		LogError(t, err)

		assert.Nil(t, b)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: dependency testtypes.InterfaceA: scoped service must be resolved from a child scope")
	})

	t.Run("slice service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[[]testtypes.InterfaceA](ctx, c)

		want := []testtypes.InterfaceA{
			&testtypes.StructA{},
			&testtypes.StructA{},
		}
		assert.Equal(t, want, got)
		assert.NoError(t, err)
	})

	t.Run("slice service values", func(t *testing.T) {
		a1 := &testtypes.StructA{}
		a2 := &testtypes.StructA{}
		want := []*testtypes.StructA{a1, a2}

		c, err := di.NewContainer(
			di.WithService(a1),
			di.WithService(a2),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[[]*testtypes.StructA](ctx, c)

		assert.Exactly(t, want, got)
		assert.NoError(t, err)
	})

	t.Run("slice service dependency", func(t *testing.T) {
		want := []testtypes.InterfaceA{
			&testtypes.StructA{},
			&testtypes.StructA{},
		}

		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func(aa []testtypes.InterfaceA) testtypes.InterfaceB {
				assert.Equal(t, want, aa)
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
		want := []testtypes.InterfaceA{
			&testtypes.StructA{},
			&testtypes.StructA{},
		}

		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB),
			di.WithService(func(b testtypes.InterfaceB, aa ...testtypes.InterfaceA) testtypes.InterfaceD {
				assert.Equal(t, &testtypes.StructB{}, b)
				assert.Equal(t, want, aa)
				return &testtypes.StructD{}
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		d, err := di.Resolve[testtypes.InterfaceD](ctx, c)
		assert.Equal(t, &testtypes.StructD{}, d)
		assert.NoError(t, err)
	})

	t.Run("alias override type", func(t *testing.T) {
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
		assert.ErrorIs(t, err, di.ErrServiceNotRegistered)
	})

	t.Run("alias same instance", func(t *testing.T) {
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

	t.Run("tag with func", func(t *testing.T) {
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
		assert.NotNil(t, a2)
		assert.NoError(t, err)
		assert.Same(t, a1, a2)
	})

	t.Run("tag with value", func(t *testing.T) {
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

	t.Run("tag with alias", func(t *testing.T) {
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
		assert.NotNil(t, got)
		assert.NoError(t, err)
	})

	t.Run("tags mixed", func(t *testing.T) {
		a1 := &testtypes.StructA{Tag: 1}
		a2 := &testtypes.StructA{Tag: 2}

		c, err := di.NewContainer(
			di.WithService(a1, di.As[testtypes.InterfaceA]()),
			di.WithService(a2, di.As[testtypes.InterfaceA](), di.WithTag(2)),
		)
		require.NoError(t, err)

		// Should we make it so that the service registered with
		// no tag takes precedence if when requesting the service with no tag?

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Same(t, a2, got, "should get a2 because it was registered last")
		assert.NoError(t, err)

		got, err = di.Resolve[testtypes.InterfaceA](ctx, c, di.WithTag(2))
		assert.Same(t, a2, got)
		assert.NoError(t, err)
	})

	t.Run("tag not registered", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithTag("tag")),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithTag("other"))
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA (Tag other): service not registered")
		assert.ErrorIs(t, err, di.ErrServiceNotRegistered)
	})

	t.Run("tagged", func(t *testing.T) {
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

	t.Run("tagged multiple", func(t *testing.T) {
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
				return nil, stderrors.New("constructor error")
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
			di.WithService(func(ctx context.Context, scope di.Scope) *TestFactory {
				// We cannot call Resolve on the scope here.
				a, err := di.Resolve[testtypes.InterfaceA](ctx, scope)
				LogError(t, err)

				assert.Nil(t, a)
				assert.EqualError(t, err,
					"di.Container.Resolve testtypes.InterfaceA: "+
						"not supported within service constructor function")

				// Contains can be called though
				hasA := scope.Contains(reflect.TypeFor[testtypes.InterfaceA]())
				assert.True(t, hasA)

				// We have to store it and we can call Resolve later.
				return NewTestFactory(scope, func(ctx context.Context, s di.Scope) testtypes.InterfaceA {
					a, err := di.Resolve[testtypes.InterfaceA](ctx, s)
					assert.NotNil(t, a)
					assert.NoError(t, err)
					return a
				})
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		factory, err := di.Resolve[*TestFactory](ctx, c)
		require.NoError(t, err)

		a := factory.Build(ctx)
		assert.NotNil(t, a)
	})

	t.Run("decorator", func(t *testing.T) {
		a1 := &testtypes.StructA{}
		calls := 0

		c, err := di.NewContainer(
			di.WithService(func() *testtypes.StructA { return a1 }),
			di.WithDecorator(func(a *testtypes.StructA) *testtypes.StructA {
				calls++
				a.Tag = "decorated"
				return a
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[*testtypes.StructA](ctx, c)

		assert.Same(t, a1, got)
		assert.Equal(t, &testtypes.StructA{Tag: "decorated"}, got)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("decorator value service", func(t *testing.T) {
		a := &testtypes.StructA{Tag: "original"}

		c, err := di.NewContainer(
			di.WithService(a, di.As[testtypes.InterfaceA]()),
			di.WithDecorator(func(testtypes.InterfaceA) testtypes.InterfaceA {
				assert.Fail(t, "decorator should not be called")
				return nil
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Same(t, a, got, "value services cannot be decorated")
		assert.NoError(t, err)
	})

	t.Run("decorators multiple", func(t *testing.T) {
		a1 := &testtypes.StructA{Tag: 1}
		a2 := &testtypes.StructA{Tag: 2}
		a3 := &testtypes.StructA{Tag: 3}
		calls1 := 0
		calls2 := 0

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return a1 }),
			di.WithDecorator(func(a testtypes.InterfaceA) testtypes.InterfaceA {
				assert.Same(t, a1, a)
				calls1++
				return a2
			}),
			di.WithDecorator(func(a testtypes.InterfaceA) testtypes.InterfaceA {
				assert.Same(t, a2, a)
				calls2++
				return a3
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)

		assert.Same(t, a3, got)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls1)
		assert.Equal(t, 1, calls2)
	})

	t.Run("decorator with context", func(t *testing.T) {
		calls := 0
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithDecorator(func(ctx context.Context, a testtypes.InterfaceA) testtypes.InterfaceA {
				assert.NotNil(t, ctx)
				calls++
				return a
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.NotNil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("decorator with dependency", func(t *testing.T) {
		calls := 0
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB),
			di.WithDecorator(func(b testtypes.InterfaceB, a testtypes.InterfaceA) testtypes.InterfaceB {
				assert.NotNil(t, a)
				calls++
				return b
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceB](ctx, c)

		assert.NotNil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("decorator with di.Scope dependency", func(t *testing.T) {
		calls := 0
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithDecorator(func(a testtypes.InterfaceA, s di.Scope) testtypes.InterfaceA {
				assert.NotNil(t, a)
				require.NotNil(t, s)
				assert.True(t, s.Contains(reflect.TypeFor[testtypes.InterfaceA]()))

				calls++
				return a
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)

		assert.NotNil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("decorator with error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() (testtypes.InterfaceA, error) {
				return nil, stderrors.New("constructor error")
			}),
			di.WithService(func() testtypes.InterfaceB {
				return &testtypes.StructB{}
			}),
			di.WithDecorator(func(testtypes.InterfaceB, testtypes.InterfaceA) testtypes.InterfaceB {
				assert.Fail(t, "decorator should not be called")
				return nil
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceB: decorator func(testtypes.InterfaceB, testtypes.InterfaceA) testtypes.InterfaceB: dependency testtypes.InterfaceA: constructor error")
	})

	t.Run("decorator with nil service", func(t *testing.T) {
		calls := 0
		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return nil }),
			di.WithDecorator(func(testtypes.InterfaceA) testtypes.InterfaceA {
				calls++
				return &testtypes.StructA{}
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)

		assert.NotNil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("decorator function dependency returns nil", func(t *testing.T) {
		calls := 0
		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return nil }),
			di.WithService(func() testtypes.InterfaceB { return &testtypes.StructB{} }),
			di.WithDecorator(func(b testtypes.InterfaceB, a testtypes.InterfaceA) testtypes.InterfaceB {
				assert.Nil(t, a)
				calls++
				return b
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceB](ctx, c)

		assert.Equal(t, &testtypes.StructB{}, got)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("decorator with tag", func(t *testing.T) {
		const tag = "decorate me"

		a := &testtypes.StructA{Tag: tag}
		calls := 0

		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func() testtypes.InterfaceA { return a },
				di.As[testtypes.InterfaceA](),
				di.WithTag(tag),
			),
			di.WithDecorator(func(aa testtypes.InterfaceA) testtypes.InterfaceA {
				assert.Same(t, a, aa)
				calls++
				return nil
			}, di.WithTag(tag)),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithTag(tag))

		assert.Nil(t, got)
		assert.NotSame(t, a, got)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("decorator with tagged", func(t *testing.T) {
		const tag = "decorate me"

		a := &testtypes.StructA{Tag: tag}
		calls := 0

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return nil }),
			di.WithService(func() testtypes.InterfaceA { return a },
				di.WithTag(tag),
			),
			di.WithService(testtypes.NewInterfaceB),
			di.WithDecorator(func(aa testtypes.InterfaceA, b testtypes.InterfaceB) testtypes.InterfaceB {
				assert.Same(t, a, aa)
				assert.NotNil(t, b)
				calls++
				return b
			}, di.WithTagged[testtypes.InterfaceA](tag)),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceB](ctx, c)

		assert.NotNil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("with module", func(t *testing.T) {
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
			LogError(t, err)
			assert.EqualError(t, err, "di.Container.Resolve testtypes.InterfaceA: dependency testtypes.InterfaceB: dependency testtypes.InterfaceA: dependency cycle detected")
		}()

		go func() {
			defer wg.Done()
			_, err := di.Resolve[testtypes.InterfaceB](ctx, c)
			LogError(t, err)
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
		LogError(t, err)

		assert.EqualError(t, err, "di.Container.Close: closed already: container closed")
		assert.ErrorIs(t, err, di.ErrContainerClosed)
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
			Return(stderrors.New("err a")).
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
			Return(stderrors.New("err a")).
			Once()
		cMock := mocks.NewInterfaceCMock(t)
		cMock.EXPECT().
			Close().
			Return(stderrors.New("err c")).
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

	t.Run("func ignore close", func(t *testing.T) {
		aMock := mocks.NewInterfaceAMock(t)

		scope, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return aMock },
				di.IgnoreClose(),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		_, err = di.Resolve[testtypes.InterfaceA](ctx, scope)
		assert.NoError(t, err)

		err = scope.Close(ctx)
		assert.NoError(t, err)
	})

	t.Run("value with close", func(t *testing.T) {
		ctx := context.Background()

		aMock := mocks.NewInterfaceAMock(t)
		aMock.EXPECT().
			Close(ctx).
			Return(nil).
			Once()

		c, err := di.NewContainer(
			di.WithService(aMock,
				di.As[testtypes.InterfaceA](),
				di.WithClose(),
			),
		)
		require.NoError(t, err)

		// Value service should be close even if it is never resolved
		err = c.Close(ctx)
		assert.NoError(t, err)
	})

	t.Run("func with close func", func(t *testing.T) {
		ctx := context.Background()

		aMock := mocks.NewInterfaceAMock(t)
		aClosed := false

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return aMock },
				di.WithCloseFunc(func(context.Context, testtypes.InterfaceA) error {
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

	t.Run("value with close func", func(t *testing.T) {
		ctx := context.Background()

		aMock := mocks.NewInterfaceAMock(t)
		aClosed := false

		c, err := di.NewContainer(
			di.WithService(aMock,
				di.As[testtypes.InterfaceA](),
				di.WithCloseFunc(func(context.Context, testtypes.InterfaceA) error {
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

	t.Run("concurrent close", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		const concurrency = 10
		expectedErr := errors.Wrap(di.ErrContainerClosed, "di.Container.Close: closed already")

		// Only one call should return a nil error
		expected := make([]error, concurrency)
		for i := range concurrency - 1 {
			expected[i] = expectedErr
		}

		results := make([]error, concurrency)
		runConcurrent(concurrency, func(i int) {
			results[i] = c.Close(context.Background())
		})

		assert.ElementsMatch(t, expected, results)
	})

	t.Run("concurrent close with resolve", func(t *testing.T) {
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

	t.Run("concurrent close with new scope", func(t *testing.T) {
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
