package dihttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/dicontext"
	"github.com/sectrean/di-kit/dihttp"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/stretchr/testify/require"
)

// noopResponseWriter discards all writes, avoiding the allocations of
// httptest.ResponseRecorder in the benchmark loop. The success path of the
// middleware never touches the writer.
type noopResponseWriter struct{}

func (noopResponseWriter) Header() http.Header         { return http.Header{} }
func (noopResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (noopResponseWriter) WriteHeader(int)             {}

var noopHandler = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

func Benchmark_Middleware(b *testing.B) {
	b.Run("request scope", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(b, err)

		mw := dihttp.NewRequestScopeMiddleware(c)
		handler := mw(noopHandler)

		w := noopResponseWriter{}
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

		// Correctness check: the handler serves without hitting an error path.
		handler.ServeHTTP(w, req)

		b.ReportAllocs()
		for b.Loop() {
			handler.ServeHTTP(w, req)
		}
	})

	b.Run("with scoped service", func(b *testing.B) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.Scoped),
		)
		require.NoError(b, err)

		var resolveErr error
		mw := dihttp.NewRequestScopeMiddleware(c)
		handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			_, resolveErr = dicontext.Resolve[testtypes.InterfaceB](r.Context())
		}))

		w := noopResponseWriter{}
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

		handler.ServeHTTP(w, req)
		require.NoError(b, resolveErr)

		b.ReportAllocs()
		for b.Loop() {
			handler.ServeHTTP(w, req)
		}
	})
}
