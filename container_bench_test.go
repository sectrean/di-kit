//go:build go1.24

package di_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/stretchr/testify/require"
)

func Benchmark_NewContainer(b *testing.B) {
	optsOneFunc := []di.ContainerOption{
		di.WithService(testtypes.NewInterfaceAStruct),
	}
	optsTwoFunc := []di.ContainerOption{
		di.WithService(testtypes.NewInterfaceAStruct),
		di.WithService(testtypes.NewInterfaceBStruct),
	}
	optsOneValue := []di.ContainerOption{
		di.WithService(&testtypes.StructA{}),
	}

	b.Run("1 func service", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = di.NewContainer(optsOneFunc...)
		}
	})

	b.Run("2 func services", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = di.NewContainer(optsTwoFunc...)
		}
	})

	b.Run("1 value service", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = di.NewContainer(optsOneValue...)
		}
	})
}

func Benchmark_Container_NewScope(b *testing.B) {
	b.Run("no new services", func(b *testing.B) {
		root, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceAStruct),
			di.WithService(testtypes.NewInterfaceBStruct, di.Scoped),
		)
		require.NoError(b, err)

		b.ReportAllocs()
		for b.Loop() {
			_, _ = root.NewScope()
		}
	})

	b.Run("1 value service", func(b *testing.B) {
		root, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)

		opts := []di.ContainerOption{
			di.WithService(&testtypes.StructA{}),
		}

		b.ReportAllocs()
		for b.Loop() {
			_, _ = root.NewScope(opts...)
		}
	})

	b.Run("1 func service", func(b *testing.B) {
		root, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)

		opts := []di.ContainerOption{
			di.WithService(testtypes.NewInterfaceB),
		}

		b.ReportAllocs()
		for b.Loop() {
			_, _ = root.NewScope(opts...)
		}
	})
}

func Benchmark_Container_Contains(b *testing.B) {
	b.Run("func", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)
		require.True(b, c.Contains(testtypes.TypeInterfaceA))

		b.ReportAllocs()
		for b.Loop() {
			_ = c.Contains(testtypes.TypeInterfaceA)
		}
	})

	b.Run("func tagged", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceA, di.WithTag("b")),
		)
		require.NoError(b, err)

		tagOpt := di.WithTag("b")
		require.True(b, c.Contains(testtypes.TypeInterfaceA, tagOpt))

		b.ReportAllocs()
		for b.Loop() {
			_ = c.Contains(testtypes.TypeInterfaceA, tagOpt)
		}
	})

	b.Run("value", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}),
		)
		require.NoError(b, err)
		require.True(b, c.Contains(testtypes.TypeStructAPtr))

		b.ReportAllocs()
		for b.Loop() {
			_ = c.Contains(testtypes.TypeStructAPtr)
		}
	})

	b.Run("value tagged", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}),
			di.WithService(&testtypes.StructA{}, di.WithTag("b")),
		)
		require.NoError(b, err)

		tagOpt := di.WithTag("b")
		require.True(b, c.Contains(testtypes.TypeStructAPtr, tagOpt))

		b.ReportAllocs()
		for b.Loop() {
			_ = c.Contains(testtypes.TypeStructAPtr, tagOpt)
		}
	})

	b.Run("not found", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)
		require.False(b, c.Contains(testtypes.TypeInterfaceB))

		b.ReportAllocs()
		for b.Loop() {
			_ = c.Contains(testtypes.TypeInterfaceB)
		}
	})
}

func Benchmark_Container_Resolve(b *testing.B) {
	ctx := context.Background()

	b.Run("value", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.StructA{}),
		)
		require.NoError(b, err)

		got, err := c.Resolve(ctx, testtypes.TypeStructA)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		for b.Loop() {
			_, _ = c.Resolve(ctx, testtypes.TypeStructA)
		}
	})

	b.Run("singleton", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.Singleton),
		)
		require.NoError(b, err)

		got, err := c.Resolve(ctx, testtypes.TypeInterfaceA)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		for b.Loop() {
			_, _ = c.Resolve(ctx, testtypes.TypeInterfaceA)
		}
	})

	b.Run("singleton from child scope", func(b *testing.B) {
		parent := newParent(b)
		scope, err := parent.NewScope()
		require.NoError(b, err)

		got, err := scope.Resolve(ctx, testtypes.TypeInterfaceA)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		for b.Loop() {
			_, _ = scope.Resolve(ctx, testtypes.TypeInterfaceA)
		}
	})

	b.Run("singleton from child scope parallel", func(b *testing.B) {
		parent := newParent(b)
		scope, err := parent.NewScope()
		require.NoError(b, err)

		got, err := scope.Resolve(ctx, testtypes.TypeInterfaceA)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = scope.Resolve(ctx, testtypes.TypeInterfaceA)
			}
		})
	})

	b.Run("scoped first call", func(b *testing.B) {
		parent := newParent(b)
		// Warm the singleton dependency so it isn't reconstructed each iteration.
		_, err := parent.Resolve(ctx, testtypes.TypeInterfaceA)
		require.NoError(b, err)

		check, err := parent.NewScope()
		require.NoError(b, err)
		got, err := check.Resolve(ctx, testtypes.TypeInterfaceB)
		require.NoError(b, err)
		require.NotNil(b, got)

		// Each iteration must resolve on a fresh scope to hit the construction path,
		// so scopes are pre-allocated and indexed (b.Loop does not expose b.N).
		scopes := newChildScopes(b, parent)
		b.ReportAllocs()
		b.ResetTimer()

		for i := range b.N {
			_, _ = scopes[i].Resolve(ctx, testtypes.TypeInterfaceB)
		}
	})

	b.Run("scoped cached", func(b *testing.B) {
		parent := newParent(b)
		scope, err := parent.NewScope()
		require.NoError(b, err)

		got, err := scope.Resolve(ctx, testtypes.TypeInterfaceB)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		for b.Loop() {
			_, _ = scope.Resolve(ctx, testtypes.TypeInterfaceB)
		}
	})

	b.Run("scoped parallel", func(b *testing.B) {
		parent := newParent(b)
		scope, err := parent.NewScope()
		require.NoError(b, err)

		// Warm the cache so this measures concurrent cache hits.
		got, err := scope.Resolve(ctx, testtypes.TypeInterfaceB)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = scope.Resolve(ctx, testtypes.TypeInterfaceB)
			}
		})
	})

	b.Run("transient", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceAStruct, di.Transient),
		)
		require.NoError(b, err)

		got, err := c.Resolve(ctx, testtypes.TypeInterfaceA)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		for b.Loop() {
			_, _ = c.Resolve(ctx, testtypes.TypeInterfaceA)
		}
	})

	b.Run("transient parallel", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceAStruct, di.Transient),
		)
		require.NoError(b, err)

		got, err := c.Resolve(ctx, testtypes.TypeInterfaceA)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = c.Resolve(ctx, testtypes.TypeInterfaceA)
			}
		})
	})

	b.Run("transient with deps", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceAStruct, di.Transient),
			di.WithService(testtypes.NewInterfaceBStruct, di.Transient),
		)
		require.NoError(b, err)

		got, err := c.Resolve(ctx, testtypes.TypeInterfaceB)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		for b.Loop() {
			_, _ = c.Resolve(ctx, testtypes.TypeInterfaceB)
		}
	})

	b.Run("deep chain", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceAStruct, di.Transient),
			di.WithService(testtypes.NewInterfaceBStruct, di.Transient),
			di.WithService(testtypes.NewInterfaceCStruct, di.Transient),
			di.WithService(testtypes.NewInterfaceDStruct, di.Transient),
		)
		require.NoError(b, err)

		got, err := c.Resolve(ctx, testtypes.TypeInterfaceD)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		for b.Loop() {
			_, _ = c.Resolve(ctx, testtypes.TypeInterfaceD)
		}
	})

	b.Run("slice of 3", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)

		sliceType := reflect.TypeFor[[]testtypes.InterfaceA]()
		got, err := c.Resolve(ctx, sliceType)
		require.NoError(b, err)
		require.Len(b, got, 3)

		b.ReportAllocs()
		for b.Loop() {
			_, _ = c.Resolve(ctx, sliceType)
		}
	})

	b.Run("tagged", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithTag("x")),
		)
		require.NoError(b, err)

		tagOpt := di.WithTag("x")
		got, err := c.Resolve(ctx, testtypes.TypeInterfaceA, tagOpt)
		require.NoError(b, err)
		require.NotNil(b, got)

		b.ReportAllocs()
		for b.Loop() {
			_, _ = c.Resolve(ctx, testtypes.TypeInterfaceA, tagOpt)
		}
	})
}

func Benchmark_Resolve(b *testing.B) {
	c, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceA),
	)
	require.NoError(b, err)

	ctx := context.Background()
	got, err := di.Resolve[testtypes.InterfaceA](ctx, c)
	require.NoError(b, err)
	require.NotNil(b, got)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = di.Resolve[testtypes.InterfaceA](ctx, c)
	}
}

func Benchmark_MustResolve(b *testing.B) {
	c, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceA),
	)
	require.NoError(b, err)

	ctx := context.Background()
	require.NotNil(b, di.MustResolve[testtypes.InterfaceA](ctx, c))

	b.ReportAllocs()
	for b.Loop() {
		_ = di.MustResolve[testtypes.InterfaceA](ctx, c)
	}
}

func Benchmark_Invoke(b *testing.B) {
	ctx := context.Background()

	b.Run("1 param", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)

		fn := func(testtypes.InterfaceA) {}
		require.NoError(b, di.Invoke(ctx, c, fn))

		b.ReportAllocs()
		for b.Loop() {
			_ = di.Invoke(ctx, c, fn)
		}
	})

	b.Run("3 params", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB),
			di.WithService(testtypes.NewInterfaceC),
		)
		require.NoError(b, err)

		fn := func(testtypes.InterfaceA, testtypes.InterfaceB, testtypes.InterfaceC) {}
		require.NoError(b, di.Invoke(ctx, c, fn))

		b.ReportAllocs()
		for b.Loop() {
			_ = di.Invoke(ctx, c, fn)
		}
	})

	b.Run("context.Context param", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)

		fn := func(context.Context, testtypes.InterfaceA) {}
		require.NoError(b, di.Invoke(ctx, c, fn))

		b.ReportAllocs()
		for b.Loop() {
			_ = di.Invoke(ctx, c, fn)
		}
	})

	b.Run("di.WithTagged", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.WithTag("x")),
		)
		require.NoError(b, err)

		fn := func(testtypes.InterfaceA) {}
		taggedOpt := di.WithTagged[testtypes.InterfaceA]("x")
		require.NoError(b, di.Invoke(ctx, c, fn, taggedOpt))

		b.ReportAllocs()
		for b.Loop() {
			_ = di.Invoke(ctx, c, fn, taggedOpt)
		}
	})
}

func Benchmark_Container_Close(b *testing.B) {
	// This benchmark usually times out because the setup takes so long and the Close operation is very cheap.
	// The number of iterations keeps increasing until the benchmark times out, so we skip it to avoid wasting time.
	b.Skip("Timeout")

	ctx := context.Background()

	// Correctness check: the container resolves the full closer chain.
	check := newClosableContainer(b)
	got, err := check.Resolve(ctx, testtypes.TypeInterfaceD)
	require.NoError(b, err)
	require.NotNil(b, got)
	require.NoError(b, check.Close(ctx))

	// Close is one-shot, so each iteration needs a freshly resolved container.
	// Pre-build them (b.Loop does not expose b.N).
	containers := make([]*di.Container, b.N)
	for i := range b.N {
		c := newClosableContainer(b)
		_, _ = c.Resolve(ctx, testtypes.TypeInterfaceD)
		containers[i] = c
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		_ = containers[i].Close(ctx)
	}
}

func newParent(b *testing.B) *di.Container {
	parent, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceAStruct, di.Singleton),
		di.WithService(testtypes.NewInterfaceBStruct, di.Scoped),
	)
	require.NoError(b, err)
	return parent
}

func newChildScopes(b *testing.B, parent *di.Container) []*di.Container {
	scopes := make([]*di.Container, b.N)
	for i := range b.N {
		scopes[i], _ = parent.NewScope()
	}
	return scopes
}

// newClosableContainer builds a container with the full A→B→C→D singleton chain,
// so resolving InterfaceD registers four closers with distinct Close signatures.
func newClosableContainer(b *testing.B) *di.Container {
	c, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceA),
		di.WithService(testtypes.NewInterfaceB),
		di.WithService(testtypes.NewInterfaceC),
		di.WithService(testtypes.NewInterfaceD),
	)
	require.NoError(b, err)
	return c
}
