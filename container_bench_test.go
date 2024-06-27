package di_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/testtypes"
	"github.com/stretchr/testify/require"
)

// TODO: Re-organize the benchmarks to use more sub-tests.

func Benchmark_NewContainer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB),
		)
	}
}

func Benchmark_Container_NewScope(b *testing.B) {
	root, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceA),
		di.WithService(testtypes.NewInterfaceB, di.Scoped),
	)
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = root.NewScope()
	}
}

func Benchmark_Container_Contains(b *testing.B) {
	c, err := di.NewContainer(
		di.WithService(&testtypes.StructA{}),
	)
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Contains(reflect.TypeFor[testtypes.InterfaceA]())
	}
}

func Benchmark_Container_Contains_WithTag(b *testing.B) {
	c, err := di.NewContainer(
		di.WithService(&testtypes.StructA{}),
		di.WithService(&testtypes.StructA{}, di.WithTag("b")),
	)
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Contains(reflect.TypeFor[testtypes.InterfaceA](), di.WithTag("b"))
	}
}

func Benchmark_Container_Resolve_OneValueService(b *testing.B) {
	c, err := di.NewContainer(
		di.WithService(&testtypes.StructA{}),
	)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[*testtypes.StructA](ctx, c)
	}
}

func Benchmark_Container_Resolve_OneValueStruct(b *testing.B) {
	c, err := di.NewContainer(
		di.WithService(testtypes.StructA{}),
	)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[testtypes.StructA](ctx, c)
	}
}

func Benchmark_Container_Resolve_OneFunc_Singleton(b *testing.B) {
	c, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceA, di.Singleton),
	)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[testtypes.InterfaceA](ctx, c)
	}
}

func Benchmark_Container_Resolve_Scopes(b *testing.B) {
	var newParent = func(b *testing.B) *di.Container {
		parent, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.Singleton),
			di.WithService(testtypes.NewInterfaceB, di.Scoped),
		)
		require.NoError(b, err)
		return parent
	}

	var newChildScopes = func(b *testing.B, parent *di.Container) []*di.Container {
		scopes := make([]*di.Container, b.N)
		for i := 0; i < b.N; i++ {
			scopes[i], _ = parent.NewScope()
		}
		return scopes
	}

	b.Run("create child scope", func(b *testing.B) {
		parent := newParent(b)

		for i := 0; i < b.N; i++ {
			_, _ = parent.NewScope()
		}
	})

	b.Run("resolve singleton", func(b *testing.B) {
		ctx := context.Background()
		parent := newParent(b)
		scope, _ := parent.NewScope()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = di.Resolve[testtypes.InterfaceA](ctx, scope)
		}
	})

	b.Run("resolve singleton parallel", func(b *testing.B) {
		ctx := context.Background()
		parent := newParent(b)
		scope, _ := parent.NewScope()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = di.Resolve[testtypes.InterfaceA](ctx, scope)
			}
		})
	})

	b.Run("resolve scoped first time", func(b *testing.B) {
		ctx := context.Background()
		parent := newParent(b)
		_, _ = di.Resolve[testtypes.InterfaceA](ctx, parent)
		scopes := newChildScopes(b, parent)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = di.Resolve[testtypes.InterfaceB](ctx, scopes[i])
		}
	})

	b.Run("resolve scoped", func(b *testing.B) {
		ctx := context.Background()
		parent := newParent(b)
		scope, _ := parent.NewScope()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = di.Resolve[testtypes.InterfaceB](ctx, scope)
		}
	})

	b.Run("resolve scoped parallel", func(b *testing.B) {
		ctx := context.Background()
		parent := newParent(b)
		scope, _ := parent.NewScope()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = di.Resolve[testtypes.InterfaceB](ctx, scope)
			}
		})
	})
}

func Benchmark_Container_Resolve_OneFunc_Transient(b *testing.B) {
	c, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceA, di.Transient),
	)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[testtypes.InterfaceA](ctx, c)
	}
}

func Benchmark_Container_Resolve_TwoFunc_Transient(b *testing.B) {
	c, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceA, di.Transient),
		di.WithService(testtypes.NewInterfaceB, di.Transient),
	)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[testtypes.InterfaceB](ctx, c)
	}
}
