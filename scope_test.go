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
