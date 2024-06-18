package di_test

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/mocks"
	"github.com/johnrutherford/di-kit/internal/testtypes"
)

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

	t.Run("with unsupported service kind", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(1234),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "new container: with service int: unsupported kind int")
	})

	t.Run("with nil value", func(t *testing.T) {
		var a testtypes.InterfaceA
		c, err := di.NewContainer(
			di.WithService(a),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "new container: with service: funcOrValue is nil")
	})

	t.Run("only options", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(di.Singleton, di.WithKey("key")),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "new container: with service di.Lifetime: unexpected RegisterOption as funcOrValue")
	})

	t.Run("func alias not assignable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.As[*testtypes.StructA]()),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "new container: with service func() testtypes.InterfaceA: as *testtypes.StructA: type testtypes.InterfaceA not assignable to *testtypes.StructA")
	})

	t.Run("value alias not assignable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}, di.As[testtypes.InterfaceB]()),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "new container: with service *testtypes.StructA: as testtypes.InterfaceB: type *testtypes.StructA not assignable to testtypes.InterfaceB")
	})

	t.Run("with keyed not found", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				di.WithKeyed[testtypes.InterfaceB]("key"),
			),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "new container: with service func() testtypes.InterfaceA: with keyed testtypes.InterfaceB: argument not found")
	})

	t.Run("value with keyed", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				// This option will be ignored.
				di.WithKeyed[testtypes.InterfaceB]("key"),
			),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "new container: with service func() testtypes.InterfaceA: with keyed testtypes.InterfaceB: argument not found")
	})

	t.Run("with close func not assingable", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				di.WithCloseFunc(func(context.Context, *testtypes.StructA) error { return nil }),
			),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "new container: with service func() testtypes.InterfaceA: with close func: service type testtypes.InterfaceA is not assignable to *testtypes.StructA")
	})

	t.Run("unsupported func signature", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() (testtypes.InterfaceA, testtypes.InterfaceB) { return nil, nil }),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err,
			"new container: with service func() (testtypes.InterfaceA, testtypes.InterfaceB): function must return Service or (Service, error)")
	})

	t.Run("multiple errors", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService([]testtypes.InterfaceA{}),
			di.WithService(testtypes.NewInterfaceA, di.As[testtypes.InterfaceB]()),
		)
		LogError(t, err)

		assert.Nil(t, c)
		assert.EqualError(t, err, "new container: with service []testtypes.InterfaceA: unsupported kind slice\nwith service func() testtypes.InterfaceA: as testtypes.InterfaceB: type testtypes.InterfaceA not assignable to testtypes.InterfaceB")
	})
}

func Test_Container_NewScope(t *testing.T) {
	t.Run("no new services", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.Scoped),
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

		assert.True(t, c.Contains(reflect.TypeFor[testtypes.InterfaceA]()))
		assert.False(t, c.Contains(reflect.TypeFor[testtypes.InterfaceB]()))

		scope, err := c.NewScope(
			di.WithService(testtypes.NewInterfaceB),
		)
		assert.NotNil(t, scope)
		assert.NoError(t, err)

		assert.True(t, scope.Contains(reflect.TypeFor[testtypes.InterfaceA]()))
		assert.True(t, scope.Contains(reflect.TypeFor[testtypes.InterfaceB]()))
	})

	t.Run("with service error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		scope, err := c.NewScope(
			di.WithService(di.Scoped),
		)
		LogError(t, err)

		assert.Nil(t, scope)
		assert.EqualError(t, err, "new scope: with service di.Lifetime: unexpected RegisterOption as funcOrValue")
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
		assert.EqualError(t, err, "new scope: container closed")
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

	t.Run("with key", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithKey("key")),
		)
		require.NoError(t, err)

		has := c.Contains(reflect.TypeFor[testtypes.InterfaceA](), di.WithKey("key"))
		assert.True(t, has)

		has = c.Contains(reflect.TypeFor[testtypes.InterfaceA]())
		assert.True(t, has)

		has = c.Contains(reflect.TypeFor[testtypes.InterfaceA](), di.WithKey("other"))
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
			di.WithService(testtypes.NewInterfaceA()),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[*testtypes.StructA](ctx, c)

		assert.Equal(t, &testtypes.StructA{}, got)
		assert.NoError(t, err)
	})

	t.Run("func service no deps", func(t *testing.T) {
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
		assert.EqualError(t, err, "resolve testtypes.InterfaceA: container closed")
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
		assert.EqualError(t, err, "resolve testtypes.InterfaceA: context canceled")
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
		assert.EqualError(t, err, "resolve testtypes.InterfaceA: context deadline exceeded")
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("service not registered", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "resolve testtypes.InterfaceA: service not registered")
		assert.ErrorIs(t, err, di.ErrServiceNotRegistered)
	})

	t.Run("di.Scope not registered", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[di.Scope](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "resolve di.Scope: service not registered")
		assert.ErrorIs(t, err, di.ErrServiceNotRegistered)
	})

	t.Run("context.Context not registered", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[context.Context](ctx, c)
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "resolve context.Context: service not registered")
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
		assert.EqualError(t, err, "resolve testtypes.InterfaceB: dependency testtypes.InterfaceA: service not registered")
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
		assert.EqualError(t, err, "resolve testtypes.InterfaceA: dependency testtypes.InterfaceB: dependency testtypes.InterfaceA: dependency cycle detected")
		assert.ErrorIs(t, err, di.ErrDependencyCycle)
	})

	t.Run("singleton lifetime", func(t *testing.T) {
		calls := 0

		c, err := di.NewContainer(
			di.WithService(
				func() testtypes.InterfaceA {
					calls++
					return &testtypes.StructA{}
				},
				di.Singleton,
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		a1, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Equal(t, a1, &testtypes.StructA{})
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)

		a2, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Same(t, a1, a2)
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("transient lifetime", func(t *testing.T) {
		calls := 0

		c, err := di.NewContainer(
			di.WithService(
				func() testtypes.InterfaceA {
					calls++
					return &testtypes.StructA{}
				},
				di.Transient,
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		a1, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Equal(t, a1, &testtypes.StructA{})
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)

		a2, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.Equal(t, a2, &testtypes.StructA{})
		assert.NoError(t, err)
		assert.Equal(t, 2, calls)
	})

	t.Run("scoped lifetime", func(t *testing.T) {
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

		for i := 0; i < 3; i++ {
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

	t.Run("slice dependency", func(t *testing.T) {
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

	t.Run("variadic dependency", func(t *testing.T) {
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

	t.Run("aliases same instance", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewStructAPtr,
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

	t.Run("func with key", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithKey("key")),
		)
		require.NoError(t, err)

		ctx := context.Background()
		a1, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithKey("key"))
		assert.NotNil(t, a1)
		assert.NoError(t, err)

		a2, err := di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.NotNil(t, a2)
		assert.NoError(t, err)
		assert.Same(t, a1, a2)
	})

	t.Run("value with key", func(t *testing.T) {
		a := &testtypes.StructA{}

		c, err := di.NewContainer(
			di.WithService(a, di.WithKey("key")),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[*testtypes.StructA](ctx, c, di.WithKey("key"))
		assert.Same(t, a, got)
		assert.NoError(t, err)
	})

	t.Run("alias with key", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewStructAPtr,
				di.As[testtypes.InterfaceA](),
				di.WithKey("key"),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithKey("key"))
		assert.NotNil(t, got)
		assert.NoError(t, err)

		got, err = di.Resolve[testtypes.InterfaceA](ctx, c)
		assert.NotNil(t, got)
		assert.NoError(t, err)
	})

	t.Run("with key not registered", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithKey("key")),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithKey("other"))
		LogError(t, err)

		assert.Nil(t, got)
		assert.EqualError(t, err, "resolve testtypes.InterfaceA (Key other): service not registered")
		assert.ErrorIs(t, err, di.ErrServiceNotRegistered)
	})

	t.Run("with keyed", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				di.WithKey("A1"),
			),
			di.WithService(func() (testtypes.InterfaceA, error) {
				return nil, stderrors.New("should not be called")
			}),
			di.WithService(func(a testtypes.InterfaceA) testtypes.InterfaceB {
				return &testtypes.StructB{}
			}, di.WithKeyed[testtypes.InterfaceA]("A1")),
		)
		require.NoError(t, err)

		ctx := context.Background()

		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		assert.Equal(t, &testtypes.StructB{}, b)
		assert.NoError(t, err)
	})

	t.Run("with keyed multiple", func(t *testing.T) {
		a1 := &testtypes.StructA{}
		a2 := &testtypes.StructA{}

		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA { return a1 }, di.WithKey("1")),
			di.WithService(func() testtypes.InterfaceA { return a2 }, di.WithKey("2")),
			di.WithService(
				func(aa2 testtypes.InterfaceA, aa1 testtypes.InterfaceA) testtypes.InterfaceB {
					assert.Same(t, a1, aa1)
					assert.Same(t, a2, aa2)
					return &testtypes.StructB{}
				},
				di.WithKeyed[testtypes.InterfaceA]("2"),
				di.WithKeyed[testtypes.InterfaceA]("1"),
			),
		)
		require.NoError(t, err)

		ctx := context.Background()
		got, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		assert.Equal(t, &testtypes.StructB{}, got)
		assert.NoError(t, err)
	})

	t.Run("constructor func error", func(t *testing.T) {
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
		assert.EqualError(t, err, "resolve testtypes.InterfaceA: constructor error")

		b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
		LogError(t, err)

		assert.Nil(t, b)
		assert.EqualError(t, err, "resolve testtypes.InterfaceB: dependency testtypes.InterfaceA: constructor error")
	})

	t.Run("context dependency", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "key", "value")

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

	t.Run("scope dependency", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(func(scope di.Scope) *Factory {
				ctx := context.Background()

				// We cannot call Resolve on the scope here.
				a, err := di.Resolve[testtypes.InterfaceA](ctx, scope)
				LogError(t, err)

				assert.Nil(t, a)
				assert.EqualError(t, err,
					"resolve testtypes.InterfaceA: "+
						"resolve not supported within constructor function for *di_test.Factory: "+
						"the di.Scope must be stored and used later")

				// Contains can be called though
				hasA := scope.Contains(reflect.TypeFor[testtypes.InterfaceA]())
				assert.True(t, hasA)

				// We have to store it and we can call Resolve later.
				return &Factory{scope: scope}
			}),
		)
		require.NoError(t, err)

		ctx := context.Background()
		factory, err := di.Resolve[*Factory](ctx, c)
		require.NoError(t, err)

		a := factory.BuildA(ctx, "arg")
		assert.NotNil(t, a)
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

		assert.EqualError(t, err, "close: already closed: container closed")
		assert.ErrorIs(t, err, di.ErrContainerClosed)
	})

	t.Run("all close funcs", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "key", "value")

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
		assert.EqualError(t, err, "close: err a")
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
		assert.EqualError(t, err, "close: err c\nerr a")
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
				di.WithCloseFunc(func(ctx context.Context, a testtypes.InterfaceA) error {
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
}
