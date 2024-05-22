package di_test

import (
	"context"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/testtypes"
	"github.com/stretchr/testify/require"
)

func BenchmarkContainer_Contains(b *testing.B) {
	c, err := di.NewContainer(
		di.Register(&testtypes.StructA{}),
	)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		_ = c.Contains(di.InterfaceAType)
	}
}

func BenchmarkContainer_Contains_WithTag(b *testing.B) {
	c, err := di.NewContainer(
		di.Register(&testtypes.StructA{}),
		di.Register(&testtypes.StructA{}, di.WithTag("b")),
	)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		_ = c.Contains(di.InterfaceAType, di.WithTag("b"))
	}
}

func BenchmarkContainer_Resolve_OneValueService(b *testing.B) {
	c, err := di.NewContainer(
		di.Register(&testtypes.StructA{}),
	)
	require.NoError(b, err)

	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[*testtypes.StructA](ctx, c)
	}
}

func BenchmarkContainer_Resolve_OneFunc_Singleton(b *testing.B) {
	c, err := di.NewContainer(
		di.Register(testtypes.NewInterfaceA, di.Singleton),
	)
	require.NoError(b, err)

	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[testtypes.InterfaceA](ctx, c)
	}
}

func BenchmarkContainer_Resolve_OneFunc_Transient(b *testing.B) {
	c, err := di.NewContainer(
		di.Register(testtypes.NewInterfaceA, di.Transient),
	)
	require.NoError(b, err)

	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[testtypes.InterfaceA](ctx, c)
	}
}

func BenchmarkContainer_Resolve_TwoFunc_Transient(b *testing.B) {
	c, err := di.NewContainer(
		di.Register(testtypes.NewInterfaceA, di.Transient),
		di.Register(testtypes.NewInterfaceB, di.Transient),
	)
	require.NoError(b, err)

	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[testtypes.InterfaceB](ctx, c)
	}
}

func BenchmarkContainer_Resolve_Concurrent(b *testing.B) {
	c, err := di.NewContainer(
		di.Register(testtypes.NewInterfaceA, di.Transient),
		di.Register(testtypes.NewInterfaceB, di.Transient),
		di.Register(testtypes.NewInterfaceC, di.Transient),
	)
	require.NoError(b, err)

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = di.Resolve[testtypes.InterfaceC](ctx, c)
		}
	})
}
