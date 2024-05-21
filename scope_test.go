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
		RunAndReturn(func(ctx context.Context, _ reflect.Type, _ ...ResolveOption) (any, error) {
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
	assert.EqualError(t, err, "invoke error")
}

func TestInvoke_NotFunc(t *testing.T) {
	scope := newScopeMock(t)

	ctx := context.Background()
	err := Invoke(ctx, scope, 1234)
	assert.EqualError(t, err, "invoke int: fn must be a function")
}

func TestInvoke_ResolveError(t *testing.T) {
	scope := newScopeMock(t)
	scope.EXPECT().
		Resolve(mock.Anything, InterfaceAType).
		Return(nil, errors.New("resolve error"))

	ctx := context.Background()
	err := Invoke(ctx, scope, func(testtypes.InterfaceA) {})
	assert.EqualError(t, err, "invoke func(testtypes.InterfaceA): resolve error")
}

func TestInvoke_ContextError(t *testing.T) {
	scope := newScopeMock(t)
	scope.EXPECT().
		Resolve(mock.Anything, reflect.TypeFor[context.Context]()).
		RunAndReturn(func(ctx context.Context, _ reflect.Type, _ ...ResolveOption) (any, error) {
			return ctx, nil
		})

	ctx := ContextCanceled()
	err := Invoke(ctx, scope, func(context.Context) {})
	assert.EqualError(t, err, "invoke func(context.Context): context canceled")
}

func TestInvoke_NoError(t *testing.T) {
	scope := newScopeMock(t)
	scope.EXPECT().
		Resolve(mock.Anything, InterfaceAType).
		Return(&testtypes.StructA{}, nil)

	ctx := context.Background()
	err := Invoke(ctx, scope, func(testtypes.InterfaceA) {})
	assert.NoError(t, err)
}

func TestMustResolve(t *testing.T) {
	scope := newScopeMock(t)
	scope.EXPECT().
		Resolve(mock.Anything, reflect.TypeFor[testtypes.InterfaceA]()).
		Return(&testtypes.StructA{}, nil)

	ctx := context.Background()
	got := MustResolve[testtypes.InterfaceA](ctx, scope)
	assert.Equal(t, &testtypes.StructA{}, got)
}

func TestMustResolve_Panic(t *testing.T) {
	scope := newScopeMock(t)
	scope.EXPECT().
		Resolve(mock.Anything, reflect.TypeFor[testtypes.InterfaceA]()).
		Return(nil, errors.New("resolve error"))

	ctx := context.Background()
	assert.PanicsWithError(t, "resolve error", func() {
		MustResolve[testtypes.InterfaceA](ctx, scope)
	})
}
