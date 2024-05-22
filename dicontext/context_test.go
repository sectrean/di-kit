package dicontext_test

import (
	"context"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/dicontext"
	"github.com/johnrutherford/di-kit/internal/testtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScope(t *testing.T) {
	c, err := di.NewContainer()
	require.NoError(t, err)

	ctx := dicontext.WithScope(context.Background(), c)
	scope := dicontext.Scope(ctx)

	assert.Same(t, c, scope)
}

func TestScope_NoScope(t *testing.T) {
	ctx := context.Background()
	scope := dicontext.Scope(ctx)
	assert.Nil(t, scope)
}

func TestResolve(t *testing.T) {
	c, err := di.NewContainer(
		di.Register(testtypes.NewInterfaceA),
	)
	require.NoError(t, err)

	ctx := dicontext.WithScope(context.Background(), c)

	got, err := dicontext.Resolve[testtypes.InterfaceA](ctx)
	assert.Equal(t, &testtypes.StructA{}, got)
	assert.NoError(t, err)
}

func TestResolve_NoScope(t *testing.T) {
	ctx := context.Background()

	got, err := dicontext.Resolve[testtypes.InterfaceA](ctx)
	assert.Nil(t, got)
	assert.EqualError(t, err,
		"resolve testtypes.InterfaceA from context: scope not found on context")
}

func TestMustResolve(t *testing.T) {
	c, err := di.NewContainer(
		di.Register(testtypes.NewInterfaceA),
	)
	require.NoError(t, err)

	ctx := dicontext.WithScope(context.Background(), c)

	got := dicontext.MustResolve[testtypes.InterfaceA](ctx)
	assert.Equal(t, &testtypes.StructA{}, got)
}

func TestMustResolve_NoScope(t *testing.T) {
	ctx := context.Background()

	assert.PanicsWithError(t, "resolve testtypes.InterfaceA from context: scope not found on context", func() {
		_ = dicontext.MustResolve[testtypes.InterfaceA](ctx)
	})
}

func TestMustResolve_Error(t *testing.T) {
	ctx := context.Background()

	assert.PanicsWithError(t, "resolve testtypes.InterfaceA from context: scope not found on context", func() {
		_ = dicontext.MustResolve[testtypes.InterfaceA](ctx)
	})
}
