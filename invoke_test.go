package di

import (
	"context"
	"reflect"
	"testing"

	"github.com/johnrutherford/di-kit/internal/errors"
	"github.com/johnrutherford/di-kit/internal/testtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInvoke(t *testing.T) {
	scope := newScopeMock(t)
	scope.EXPECT().
		Resolve(mock.Anything, reflect.TypeFor[context.Context]()).
		RunAndReturn(func(ctx context.Context, _ reflect.Type, _ ...ServiceOption) (any, error) {
			return ctx, nil
		})
	scope.EXPECT().
		Resolve(mock.Anything, reflect.TypeFor[testtypes.InterfaceA]()).
		Return(&testtypes.StructA{}, nil)
	scope.EXPECT().
		Resolve(mock.Anything, reflect.TypeFor[testtypes.InterfaceB]()).
		Return(&testtypes.StructB{}, nil)

	ctx := context.Background()
	err := Invoke(ctx, scope, func(ctx context.Context, depA testtypes.InterfaceA, depB testtypes.InterfaceB) error {
		assert.Equal(t, context.Background(), ctx)
		assert.NotNil(t, depA)
		assert.NotNil(t, depB)
		return nil
	})
	LogError(t, err)
	assert.NoError(t, err)
}

func TestInvoke_Error(t *testing.T) {
	scope := newScopeMock(t)
	scope.EXPECT().
		Resolve(mock.Anything, reflect.TypeFor[testtypes.InterfaceA]()).
		Return(&testtypes.StructA{}, nil)

	ctx := context.Background()
	err := Invoke(ctx, scope, func(testtypes.InterfaceA) error {
		return errors.New("invoke error")
	})
	LogError(t, err)
	assert.EqualError(t, err, "invoke error")
}

func TestInvoke_NotFunc(t *testing.T) {
	scope := newScopeMock(t)

	ctx := context.Background()
	err := Invoke(ctx, scope, 1234)
	LogError(t, err)
	assert.EqualError(t, err, "invoke int: fn must be a function")
}

func TestInvoke_ResolveError(t *testing.T) {
	scope := newScopeMock(t)
	scope.EXPECT().
		Resolve(mock.Anything, InterfaceAType).
		Return(nil, errors.New("resolve error"))

	ctx := context.Background()
	err := Invoke(ctx, scope, func(testtypes.InterfaceA) {})
	LogError(t, err)
	assert.EqualError(t, err, "invoke func(testtypes.InterfaceA): resolve error")
}

func TestInvoke_ContextError(t *testing.T) {
	scope := newScopeMock(t)
	scope.EXPECT().
		Resolve(mock.Anything, reflect.TypeFor[context.Context]()).
		RunAndReturn(func(ctx context.Context, _ reflect.Type, _ ...ServiceOption) (any, error) {
			return ctx, nil
		})

	ctx := ContextCanceled()
	err := Invoke(ctx, scope, func(context.Context) {})
	LogError(t, err)
	assert.EqualError(t, err, "invoke func(context.Context): context canceled")
}

func TestInvoke_NoError(t *testing.T) {
	scope := newScopeMock(t)
	scope.EXPECT().
		Resolve(mock.Anything, InterfaceAType).
		Return(&testtypes.StructA{}, nil)

	ctx := context.Background()
	err := Invoke(ctx, scope, func(testtypes.InterfaceA) {})
	LogError(t, err)
	assert.NoError(t, err)
}

func TestInvoke_WithKeyed(t *testing.T) {
	scope := newScopeMock(t)
	scope.EXPECT().
		Resolve(mock.Anything, InterfaceAType, WithKey("key")).
		Return(&testtypes.StructA{}, nil)

	ctx := context.Background()
	err := Invoke(ctx, scope,
		func(testtypes.InterfaceA) {},
		WithKeyed[testtypes.InterfaceA]("key"),
	)
	LogError(t, err)
	assert.NoError(t, err)
}

func TestInvoke_WithKeyed_DepNotFound(t *testing.T) {
	scope := newScopeMock(t)

	ctx := context.Background()
	err := Invoke(ctx, scope, func() {}, WithKeyed[testtypes.InterfaceA]("key"))
	LogError(t, err)
	assert.EqualError(t, err, "invoke func(): with keyed testtypes.InterfaceA: argument not found")
}
