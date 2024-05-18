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
		scope, err := di.NewContainer(di.WithParent(root))
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

func TestServicesWithTags(t *testing.T) {
	a1 := &testtypes.StructA{}
	a2 := &testtypes.StructA{}

	c, err := di.NewContainer(
		di.WithService(func() testtypes.InterfaceA { return a1 }, di.WithTag("1")),
		di.WithService(func() testtypes.InterfaceA { return a2 }, di.WithTag("2")),
	)
	require.NoError(t, err)

	ctx := context.Background()
	got1, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithTag("1"))
	assert.Exactly(t, a1, got1)
	assert.NoError(t, err)

	got2, err := di.Resolve[testtypes.InterfaceA](ctx, c, di.WithTag("2"))
	assert.Exactly(t, a2, got2)
	assert.NoError(t, err)

	slice, err := di.Resolve[[]testtypes.InterfaceA](ctx, c)

	want := []testtypes.InterfaceA{a1, a2}
	assert.Equal(t, want, slice)
	assert.NoError(t, err)
}
