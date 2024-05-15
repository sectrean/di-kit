package di_test

import (
	"context"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/testtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Test constructor functions with errors
// TODO: Test tags
// TODO: Test aliases
// TODO: Test closers

func TestSingleton(t *testing.T) {
	calls := 0
	a := &testtypes.StructA{}

	c, err := di.NewContainer(
		di.RegisterFunc(
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
		di.RegisterFunc(
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
		di.RegisterFunc(
			func() testtypes.InterfaceA {
				aCalls++
				return a
			},
			di.Singleton,
		),
		di.RegisterFunc(
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
