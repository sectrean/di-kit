package di_test

import (
	"context"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/testtypes"
	"github.com/stretchr/testify/require"
)

func BenchmarkNewContainer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = di.NewContainer(
			di.Register(testtypes.NewInterfaceA),
			di.Register(testtypes.NewInterfaceB),
		)
	}
}

func BenchmarkNewContainer_WithParent(b *testing.B) {
	root, err := di.NewContainer(
		di.Register(testtypes.NewInterfaceA),
		di.Register(testtypes.NewInterfaceB, di.Scoped),
	)
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = di.NewContainer(di.WithParent(root))
	}
}

func BenchmarkContainer_Contains(b *testing.B) {
	c, err := di.NewContainer(
		di.Register(&testtypes.StructA{}),
	)
	require.NoError(b, err)

	b.ResetTimer()

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

	b.ResetTimer()

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

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[*testtypes.StructA](ctx, c)
	}
}

func BenchmarkContainer_Resolve_OneValueStruct(b *testing.B) {
	c, err := di.NewContainer(
		di.Register(testtypes.StructA{}),
	)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[testtypes.StructA](ctx, c)
	}
}

func BenchmarkContainer_Resolve_OneFunc_Singleton(b *testing.B) {
	c, err := di.NewContainer(
		di.Register(testtypes.NewInterfaceA, di.Singleton),
	)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[testtypes.InterfaceA](ctx, c)
	}
}

func BenchmarkContainer_Resolve_Scopes(b *testing.B) {
	parent, err := di.NewContainer(
		di.Register(testtypes.NewInterfaceA, di.Singleton),
		di.Register(testtypes.NewInterfaceB, di.Scoped),
	)
	require.NoError(b, err)

	var newChildScope = func(b *testing.B) *di.Container {
		scope, err := di.NewContainer(di.WithParent(parent))
		require.NoError(b, err)
		return scope
	}

	var newChildScopes = func(b *testing.B) []*di.Container {
		scopes := make([]*di.Container, b.N)
		for i := 0; i < b.N; i++ {
			scopes[i], err = di.NewContainer(di.WithParent(parent))
			require.NoError(b, err)
		}
		return scopes
	}

	b.Run("create child scope", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = di.NewContainer(di.WithParent(parent))
		}
	})

	b.Run("resolve singleton first time", func(b *testing.B) {
		ctx := context.Background()
		scopes := newChildScopes(b)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = di.Resolve[testtypes.InterfaceA](ctx, scopes[i])
		}
	})

	b.Run("resolve singleton", func(b *testing.B) {
		ctx := context.Background()
		scope := newChildScope(b)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = di.Resolve[testtypes.InterfaceA](ctx, scope)
		}
	})

	b.Run("resolve singleton parallel", func(b *testing.B) {
		ctx := context.Background()
		scope := newChildScope(b)
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = di.Resolve[testtypes.InterfaceA](ctx, scope)
			}
		})
	})

	b.Run("resolve scoped first time", func(b *testing.B) {
		ctx := context.Background()
		scopes := newChildScopes(b)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = di.Resolve[testtypes.InterfaceB](ctx, scopes[i])
		}
	})

	b.Run("resolve scoped", func(b *testing.B) {
		ctx := context.Background()
		scope := newChildScope(b)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = di.Resolve[testtypes.InterfaceB](ctx, scope)
		}
	})

	b.Run("resolve scoped parallel", func(b *testing.B) {
		ctx := context.Background()
		scope := newChildScope(b)
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = di.Resolve[testtypes.InterfaceB](ctx, scope)
			}
		})
	})

	// b.Run("close child scope", func(b *testing.B) {
	// 	b.StopTimer()
	// 	ctx := context.Background()
	// 	scopes := newChildScopes(b)

	// 	for i := 0; i < b.N; i++ {
	// 		_, _ = di.Resolve[testtypes.InterfaceA](ctx, scopes[i])
	// 	}

	// 	b.StartTimer()

	// 	for i := 0; i < b.N; i++ {
	// 		_ = scopes[i].Close(ctx)
	// 	}
	// })
}

func BenchmarkContainer_Resolve_OneFunc_Transient(b *testing.B) {
	c, err := di.NewContainer(
		di.Register(testtypes.NewInterfaceA, di.Transient),
	)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()

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

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = di.Resolve[testtypes.InterfaceB](ctx, c)
	}
}
