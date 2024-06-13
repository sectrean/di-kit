package di_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/errors"
	"github.com/johnrutherford/di-kit/internal/mocks"
	"github.com/johnrutherford/di-kit/internal/testtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSingleton(t *testing.T) {
	calls := 0
	a := &testtypes.StructA{}

	c, err := di.NewContainer(
		di.WithService(
			func() testtypes.InterfaceA {
				calls++
				return a
			},
			di.Singleton,
		),
	)
	require.NoError(t, err)

	ctx := context.Background()
	a1, err := di.Resolve[testtypes.InterfaceA](ctx, c)
	assert.Exactly(t, a, a1)
	assert.NoError(t, err)
	assert.Equal(t, 1, calls)

	a2, err := di.Resolve[testtypes.InterfaceA](ctx, c)
	assert.Exactly(t, a, a2)
	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestTransient(t *testing.T) {
	calls := 0
	b := &testtypes.StructB{}

	c, err := di.NewContainer(
		di.WithService(
			func() testtypes.InterfaceB {
				calls++
				return b
			},
			di.Transient,
		),
	)
	require.NoError(t, err)

	ctx := context.Background()
	b1, err := di.Resolve[testtypes.InterfaceB](ctx, c)
	assert.Exactly(t, b, b1)
	assert.NoError(t, err)
	assert.Equal(t, 1, calls)

	b2, err := di.Resolve[testtypes.InterfaceB](ctx, c)
	assert.Exactly(t, b, b2)
	assert.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestScoped(t *testing.T) {
	aCalls := 0
	bCalls := 0
	a := &testtypes.StructA{}

	root, err := di.NewContainer(
		di.WithService(
			func() testtypes.InterfaceA {
				aCalls++
				return a
			},
			di.Singleton,
		),
		di.WithService(
			func(depA testtypes.InterfaceA) testtypes.InterfaceB {
				assert.Equal(t, a, depA)
				bCalls++
				return &testtypes.StructB{}
			},
			di.Scoped,
		),
	)
	require.NoError(t, err)

	ctx := context.Background()

	for i := 0; i < 3; i++ {
		scope, err := root.NewScope()
		require.NoError(t, err)

		b, err := di.Resolve[testtypes.InterfaceB](ctx, scope)
		assert.NotNil(t, b)
		assert.NoError(t, err)

		b2, err := di.Resolve[testtypes.InterfaceB](ctx, scope)
		assert.NotNil(t, b2)
		assert.NoError(t, err)

		assert.Exactly(t, b, b2)
	}

	assert.Equal(t, 1, aCalls)
	assert.Equal(t, 3, bCalls)
}

func TestSliceService(t *testing.T) {
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
}

func TestAliases(t *testing.T) {
	c, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceA,
			di.As[testtypes.InterfaceA](),
		),
	)
	require.NoError(t, err)

	ctx := context.Background()
	got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
	assert.Equal(t, &testtypes.StructA{}, got)
	assert.NoError(t, err)

	_, err = di.Resolve[*testtypes.StructA](ctx, c)
	assert.ErrorIs(t, err, di.ErrTypeNotRegistered)
}

func TestAliases_SameInstance(t *testing.T) {
	a := &testtypes.StructA{}
	calls := 0

	c, err := di.NewContainer(
		di.WithService(
			func() *testtypes.StructA {
				calls++
				return a
			},
			di.As[testtypes.InterfaceA](),
			di.As[*testtypes.StructA](),
		),
	)
	require.NoError(t, err)

	ctx := context.Background()

	got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
	assert.Same(t, a, got)
	assert.NoError(t, err)

	got, err = di.Resolve[*testtypes.StructA](ctx, c)
	assert.Same(t, a, got)
	assert.NoError(t, err)

	assert.Equal(t, 1, calls, "constructor func should only be called once")
}

func TestFuncServiceError(t *testing.T) {
	c, err := di.NewContainer(
		di.WithService(func() (testtypes.InterfaceA, error) {
			return &testtypes.StructA{}, errors.New("constructor error")
		}),
	)
	require.NoError(t, err)

	ctx := context.Background()
	got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
	assert.Equal(t, &testtypes.StructA{}, got)
	assert.EqualError(t, err, "resolve testtypes.InterfaceA: constructor error")
}

func TestServicesWithKeys(t *testing.T) {
	a1 := &testtypes.StructA{}
	a2 := &testtypes.StructA{}

	c, err := di.NewContainer(
		di.WithService(func() testtypes.InterfaceA { return a1 }, di.WithKey("1")),
		di.WithService(func() testtypes.InterfaceA { return a2 }, di.WithKey("2")),
		di.WithService(func(aa1 testtypes.InterfaceA, aa2 testtypes.InterfaceA) testtypes.InterfaceB {
			assert.Same(t, a1, aa1)
			assert.Same(t, a2, aa2)
			return &testtypes.StructB{}
		}),
		di.WithService(testtypes.NewInterfaceC, di.WithKeyed[testtypes.InterfaceA]("blah")),
	)
	require.NoError(t, err)

	ctx := context.Background()
	got1, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithKey("1"))
	assert.Exactly(t, a1, got1)
	assert.NoError(t, err)

	got2, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithKey("2"))
	assert.Exactly(t, a2, got2)
	assert.NoError(t, err)

	slice, err := di.Resolve[[]testtypes.InterfaceA](ctx, c)

	want := []testtypes.InterfaceA{a1, a2}
	assert.Equal(t, want, slice)
	assert.NoError(t, err)
}

func TestSliceServices(t *testing.T) {
	ctx := context.Background()

	a1 := &testtypes.StructA{}
	a2 := &testtypes.StructA{}
	want := []testtypes.InterfaceA{a1, a2}

	c, err := di.NewContainer(
		di.WithService(a1, di.As[testtypes.InterfaceA]()),
		di.WithService(a2, di.As[testtypes.InterfaceA]()),
		di.WithService(func(aa []testtypes.InterfaceA) testtypes.InterfaceB {
			assert.Equal(t, want, aa)
			return &testtypes.StructB{}
		}),
		di.WithService(func(_ testtypes.InterfaceB, aa ...testtypes.InterfaceA) testtypes.InterfaceD {
			assert.Equal(t, want, aa)
			return &testtypes.StructD{}
		}),
	)
	require.NoError(t, err)

	slice, err := di.Resolve[[]testtypes.InterfaceA](ctx, c)
	assert.Equal(t, want, slice)
	assert.NoError(t, err)

	b, err := di.Resolve[testtypes.InterfaceB](ctx, c)
	assert.NotNil(t, b)
	assert.NoError(t, err)

	d, err := di.Resolve[testtypes.InterfaceD](ctx, c)
	assert.NotNil(t, d)
	assert.NoError(t, err)
}

func TestServiceWithKeys(t *testing.T) {
	a1 := &testtypes.StructA{}

	c, err := di.NewContainer(
		di.WithService(func() testtypes.InterfaceA { return a1 }, di.WithKey("B")),
		di.WithService(func() testtypes.InterfaceA { panic("shouldn't get called") }),
		di.WithService(
			func(a testtypes.InterfaceA) testtypes.InterfaceB {
				assert.Same(t, a1, a)
				return &testtypes.StructB{}
			},
			di.WithKeyed[testtypes.InterfaceA]("B"),
		),
	)
	require.NoError(t, err)

	ctx := context.Background()
	got, err := di.Resolve[testtypes.InterfaceB](ctx, c)
	assert.NotNil(t, got)
	assert.NoError(t, err)
}

func TestClosers(t *testing.T) {
	a := mocks.NewInterfaceAMock(t)
	a.EXPECT().
		Close(mock.Anything).
		Return(errors.New("err a")).
		Once()
	b := mocks.NewInterfaceBMock(t)
	b.EXPECT().
		Close(mock.Anything).
		Once()
	c := mocks.NewInterfaceCMock(t)
	c.EXPECT().
		Close().
		Return(errors.New("err c")).
		Once()
	d := mocks.NewInterfaceDMock(t)
	d.EXPECT().
		Close().
		Once()

	scope, err := di.NewContainer(
		di.WithService(func() testtypes.InterfaceA { return a }),
		di.WithService(func(testtypes.InterfaceA) testtypes.InterfaceB { return b }),
		di.WithService(func(testtypes.InterfaceB) testtypes.InterfaceC { return c }),
		di.WithService(func(testtypes.InterfaceC) testtypes.InterfaceD { return d }),
	)
	require.NoError(t, err)

	ctx := context.Background()
	_ = di.MustResolve[testtypes.InterfaceD](ctx, scope)
	_ = di.MustResolve[testtypes.InterfaceC](ctx, scope)
	_ = di.MustResolve[testtypes.InterfaceB](ctx, scope)
	_ = di.MustResolve[testtypes.InterfaceA](ctx, scope)

	err = scope.Close(ctx)
	assert.EqualError(t, err, "close container: err c\nerr a")
}

func TestNoCloser(t *testing.T) {
	a := mocks.NewInterfaceAMock(t)

	c, err := di.NewContainer(
		di.WithService(func() testtypes.InterfaceA { return a }, di.IgnoreCloser()),
	)
	require.NoError(t, err)

	ctx := context.Background()
	_ = di.MustResolve[testtypes.InterfaceA](ctx, c)

	err = c.Close(ctx)
	assert.NoError(t, err)
}

func TestWithCloser(t *testing.T) {
	a := mocks.NewInterfaceAMock(t)
	a.EXPECT().
		Close(mock.Anything).
		Return(nil).
		Once()

	c, err := di.NewContainer(
		di.WithService(a,
			di.As[testtypes.InterfaceA](),
			di.WithCloser(),
		),
	)
	require.NoError(t, err)

	ctx := context.Background()
	_ = di.MustResolve[testtypes.InterfaceA](ctx, c)

	err = c.Close(ctx)
	assert.NoError(t, err)
}

func TestWithCloseFunc(t *testing.T) {
	a := mocks.NewInterfaceAMock(t)
	calls := 0

	c, err := di.NewContainer(
		di.WithService(
			func() testtypes.InterfaceA { return a },
			di.WithCloseFunc(func(ctx context.Context, a testtypes.InterfaceA) error {
				calls++
				return nil
			}),
		),
	)
	require.NoError(t, err)

	ctx := context.Background()
	_ = di.MustResolve[testtypes.InterfaceA](ctx, c)

	err = c.Close(ctx)
	assert.NoError(t, err)

	assert.Equal(t, 1, calls)
}

func TestResolveWithEmptyContainer(t *testing.T) {
	c, err := di.NewContainer()
	require.NoError(t, err)

	ctx := context.Background()
	_, err = di.Resolve[testtypes.InterfaceA](ctx, c)
	assert.ErrorIs(t, err, di.ErrTypeNotRegistered)
}

func TestResolveWithContext(t *testing.T) {
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
}

func TestResolveWithScope(t *testing.T) {
	ctx := context.Background()

	c, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceA),
		di.WithService(func(scope di.Scope) *Factory {
			ctx := context.Background()

			// We cannot call Resolve on the scope here.
			a, err := di.Resolve[testtypes.InterfaceA](ctx, scope)
			assert.Nil(t, a)
			assert.EqualError(t, err,
				"resolve testtypes.InterfaceA: "+
					"resolve not supported on di.Scope while resolving *di_test.Factory: "+
					"the scope must be stored and used later")

			// Contains can be called though
			hasA := scope.Contains(reflect.TypeFor[testtypes.InterfaceA]())
			assert.True(t, hasA)

			// We have to store it and we can call Resolve later.
			return &Factory{scope: scope}
		}),
	)
	require.NoError(t, err)

	factory, err := di.Resolve[*Factory](ctx, c)
	require.NoError(t, err)

	a := factory.BuildA(ctx, "arg")
	assert.NotNil(t, a)
}

type Factory struct {
	scope di.Scope
}

func (f *Factory) BuildA(ctx context.Context, _ string) testtypes.InterfaceA {
	a, err := di.Resolve[testtypes.InterfaceA](ctx, f.scope)
	if err != nil {
		panic(err)
	}

	return a
}
