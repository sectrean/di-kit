package di_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/johnrutherford/di-gen"
	"github.com/johnrutherford/di-gen/internal/testtypes"
)

func TestContainer_BasicFunc_Invoke(t *testing.T) {
	c, err := di.NewContainer(
		di.Provide(testtypes.NewFooService),
	)
	assert.NoError(t, err)
	require.NotNil(t, c)

	var mockFoo *testtypes.MockFooService
	var invoked = false

	ctx := context.Background()
	err = c.Invoke(ctx, func(foo testtypes.FooService) {
		// Make sure we're getting the expected implementation type
		// Cast to the underlying mock type so we can use the mock
		require.IsType(t, &testtypes.MockFooService{}, foo)
		mockFoo = foo.(*testtypes.MockFooService)
		mockFoo.EXPECT().Foo().Once()

		foo.Foo()

		mockFoo.AssertExpectations(t)

		invoked = true
	})
	assert.NoError(t, err)
	assert.True(t, invoked)
}

func TestContainer_TypeNotRegistered(t *testing.T) {
	// Create an empty container
	c, err := di.NewContainer()
	assert.NoError(t, err)
	require.NotNil(t, c)

	ctx := context.Background()
	err = c.Invoke(ctx, func(foo testtypes.FooService) {
		assert.Fail(t, "This should never be called")
	})
	assert.ErrorIs(t, err, di.ErrTypeNotRegistered)

	// Close the container
	err = c.Close(ctx)
	assert.NoError(t, err)
}
