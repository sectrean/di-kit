package di_test

import (
	"context"
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/stretchr/testify/require"
)

func Benchmark_NewContainer(b *testing.B) {
	optsOneService := []di.ContainerOption{
		di.WithService(testtypes.NewInterfaceAStruct),
	}

	optsTwoServices := []di.ContainerOption{
		di.WithService(testtypes.NewInterfaceAStruct),
		di.WithService(testtypes.NewInterfaceBStruct),
	}

	optsOneServiceValue := []di.ContainerOption{
		di.WithService(&testtypes.StructA{}),
	}

	b.Run("func service one", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = di.NewContainer(optsOneService...)
		}
	})

	b.Run("func service two", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = di.NewContainer(optsTwoServices...)
		}
	})

	b.Run("value service one", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = di.NewContainer(optsOneServiceValue...)
		}
	})
}

func Benchmark_Container_NewScope(b *testing.B) {
	b.Run("no new services", func(b *testing.B) {
		root, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceAStruct),
			di.WithService(testtypes.NewInterfaceBStruct, di.ScopedLifetime),
		)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = root.NewScope()
		}
	})

	b.Run("new value service", func(b *testing.B) {
		root, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)

		b.ResetTimer()

		opts := []di.ContainerOption{
			di.WithService(&testtypes.StructA{}),
		}

		for i := 0; i < b.N; i++ {
			_, _ = root.NewScope(opts...)
		}
	})

	b.Run("new func service", func(b *testing.B) {
		root, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)

		b.ResetTimer()

		opts := []di.ContainerOption{
			di.WithService(testtypes.NewInterfaceB),
		}

		for i := 0; i < b.N; i++ {
			_, _ = root.NewScope(opts...)
		}
	})
}

func Benchmark_Container_Contains(b *testing.B) {
	b.Run("func service", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = c.Contains(testtypes.TypeInterfaceA)
		}
	})

	b.Run("tagged func service", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceA, di.WithTag("b")),
		)
		require.NoError(b, err)

		b.ResetTimer()

		tagOpt := di.WithTag("b")

		for i := 0; i < b.N; i++ {
			_ = c.Contains(testtypes.TypeInterfaceA, tagOpt)
		}
	})

	b.Run("value service", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}),
		)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = c.Contains(testtypes.TypeStructAPtr)
		}
	})

	b.Run("tagged value service", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(&testtypes.StructA{}),
			di.WithService(&testtypes.StructA{}, di.WithTag("b")),
		)
		require.NoError(b, err)

		b.ResetTimer()

		tagOpt := di.WithTag("b")

		for i := 0; i < b.N; i++ {
			_ = c.Contains(testtypes.TypeStructAPtr, tagOpt)
		}
	})
}

func Benchmark_Container_Resolve(b *testing.B) {
	b.Run("value service", func(b *testing.B) {
		ctx := context.Background()
		c, err := di.NewContainer(
			di.WithService(testtypes.StructA{}),
		)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = c.Resolve(ctx, testtypes.TypeStructA)
		}
	})

	b.Run("singleton func", func(b *testing.B) {
		ctx := context.Background()
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA, di.SingletonLifetime),
		)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = c.Resolve(ctx, testtypes.TypeInterfaceA)
		}
	})

	b.Run("singleton from child scope", func(b *testing.B) {
		ctx := context.Background()
		parent := newParent(b)
		scope, _ := parent.NewScope()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = scope.Resolve(ctx, testtypes.TypeInterfaceA)
		}
	})

	b.Run("singleton from child scope parallel", func(b *testing.B) {
		ctx := context.Background()
		parent := newParent(b)
		scope, _ := parent.NewScope()

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = scope.Resolve(ctx, testtypes.TypeInterfaceA)
			}
		})
	})

	b.Run("scoped func first call", func(b *testing.B) {
		ctx := context.Background()
		parent := newParent(b)
		_, _ = parent.Resolve(ctx, testtypes.TypeInterfaceA)
		scopes := newChildScopes(b, parent)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = scopes[i].Resolve(ctx, testtypes.TypeInterfaceB)
		}
	})

	b.Run("scoped func", func(b *testing.B) {
		ctx := context.Background()
		parent := newParent(b)
		scope, _ := parent.NewScope()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = scope.Resolve(ctx, testtypes.TypeInterfaceA)
		}
	})

	b.Run("scoped func parallel", func(b *testing.B) {
		ctx := context.Background()
		parent := newParent(b)
		scope, _ := parent.NewScope()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = scope.Resolve(ctx, testtypes.TypeInterfaceB)
			}
		})
	})

	b.Run("transient func", func(b *testing.B) {
		ctx := context.Background()
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceAStruct, di.TransientLifetime),
		)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = c.Resolve(ctx, testtypes.TypeInterfaceC)
		}
	})

	b.Run("transient func parallel", func(b *testing.B) {
		ctx := context.Background()
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceAStruct, di.TransientLifetime),
		)
		require.NoError(b, err)

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = c.Resolve(ctx, testtypes.TypeInterfaceC)
			}
		})
	})

	b.Run("transient func with transient dep", func(b *testing.B) {
		ctx := context.Background()
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceAStruct, di.TransientLifetime),
			di.WithService(testtypes.NewInterfaceBStruct, di.TransientLifetime),
		)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = c.Resolve(ctx, testtypes.TypeInterfaceB)
		}
	})
}

func newParent(b *testing.B) *di.Container {
	parent, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceAStruct, di.SingletonLifetime),
		di.WithService(testtypes.NewInterfaceBStruct, di.ScopedLifetime),
	)
	require.NoError(b, err)
	return parent
}

func newChildScopes(b *testing.B, parent *di.Container) []*di.Container {
	scopes := make([]*di.Container, b.N)
	for i := 0; i < b.N; i++ {
		scopes[i], _ = parent.NewScope()
	}
	return scopes
}
