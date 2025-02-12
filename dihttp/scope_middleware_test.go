package dihttp_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/dicontext"
	"github.com/sectrean/di-kit/dihttp"
	"github.com/sectrean/di-kit/internal/errors"
	"github.com/sectrean/di-kit/internal/mocks"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/sectrean/di-kit/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_NewRequestScopeMiddleware(t *testing.T) {
	t.Run("nil parent", func(t *testing.T) {
		mw, err := dihttp.NewRequestScopeMiddleware(nil)
		testutils.LogError(t, err)

		assert.Nil(t, mw)
		assert.EqualError(t, err, "dihttp.NewRequestScopeMiddleware: parent is nil")
	})

	t.Run("with new scope error handler nil", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		mw, err := dihttp.NewRequestScopeMiddleware(c,
			dihttp.WithNewScopeErrorHandler(nil),
		)
		testutils.LogError(t, err)

		assert.Nil(t, mw)
		assert.EqualError(t, err, "dihttp.NewRequestScopeMiddleware: WithNewScopeErrorHandler: h is nil")
	})

	t.Run("with scope close error handler nil", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		mw, err := dihttp.NewRequestScopeMiddleware(c,
			dihttp.WithScopeCloseErrorHandler(nil),
		)
		testutils.LogError(t, err)

		assert.Nil(t, mw)
		assert.EqualError(t, err, "dihttp.NewRequestScopeMiddleware: WithScopeCloseErrorHandler: h is nil")
	})
}

func Test_Middleware(t *testing.T) {
	t.Run("scoped service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.ScopedLifetime),
		)
		require.NoError(t, err)

		mw, err := dihttp.NewRequestScopeMiddleware(c)
		require.NoError(t, err)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			b, resolveErr := dicontext.Resolve[testtypes.InterfaceB](ctx)
			assert.NotNil(t, b)
			assert.NoError(t, resolveErr)

			w.WriteHeader(http.StatusOK)
		})

		code := RunRequest(t, mw(handler))
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("*http.Request service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		mw, err := dihttp.NewRequestScopeMiddleware(c)
		require.NoError(t, err)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			req, resolveErr := dicontext.Resolve[*http.Request](ctx)

			assert.Equal(t, r, req.WithContext(ctx))
			assert.NoError(t, resolveErr)

			w.WriteHeader(http.StatusOK)
		})

		code := RunRequest(t, mw(handler))
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("new service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		mw, err := dihttp.NewRequestScopeMiddleware(c,
			dihttp.WithContainerOptions(
				di.WithService(testtypes.NewInterfaceB),
			),
		)
		require.NoError(t, err)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			b, resolveErr := dicontext.Resolve[testtypes.InterfaceB](ctx)

			assert.NotNil(t, b)
			assert.NoError(t, resolveErr)

			w.WriteHeader(http.StatusOK)
		})

		code := RunRequest(t, mw(handler))
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("concurrent requests", func(t *testing.T) {
		const concurrency = 20
		i := atomic.Int32{}

		c, err := di.NewContainer(
			di.WithService(func(*http.Request) testtypes.InterfaceA {
				return &testtypes.StructA{
					Tag: i.Add(1),
				}
			}, di.ScopedLifetime),
		)
		require.NoError(t, err)

		mw, err := dihttp.NewRequestScopeMiddleware(c)
		require.NoError(t, err)

		results := make(chan testtypes.InterfaceA, concurrency)

		var handler http.Handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			a, resolveErr := dicontext.Resolve[testtypes.InterfaceA](ctx)
			assert.NotNil(t, a)
			assert.NoError(t, resolveErr)

			results <- a

			w.WriteHeader(http.StatusOK)
		})
		handler = mw(handler)

		testutils.RunParallel(concurrency, func(int) {
			code := RunRequest(t, handler)
			assert.Equal(t, http.StatusOK, code)
		})
		close(results)

		var res testtypes.InterfaceA
		for a := range results {
			assert.NotEqual(t, res, a, "these should all be different instances")
			res = a
		}
	})

	t.Run("new scope error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		called := false

		mw, err := dihttp.NewRequestScopeMiddleware(c,
			dihttp.WithContainerOptions(
				di.WithService(nil),
			),
			dihttp.WithNewScopeErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
				assert.NotNil(t, w)
				assert.NotNil(t, r)
				assert.EqualError(t, err, "di.Container.NewScope: WithService: funcOrValue is nil")
				called = true

				w.WriteHeader(599)
			}),
		)
		require.NoError(t, err)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Fail(t, "handler should not get called")
		})

		code := RunRequest(t, mw(handler))
		assert.Equal(t, 599, code)

		assert.True(t, called)
	})

	t.Run("close error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(func() testtypes.InterfaceA {
				a := mocks.NewInterfaceAMock(t)
				a.EXPECT().
					Close(mock.Anything).
					Return(errors.New("close error"))

				return a
			}, di.TransientLifetime),
		)
		require.NoError(t, err)

		called := false

		mw, err := dihttp.NewRequestScopeMiddleware(c,
			dihttp.WithScopeCloseErrorHandler(func(r *http.Request, err error) {
				assert.NotNil(t, r)
				assert.EqualError(t, err, "di.Container.Close: close error")
				called = true
			}),
		)
		require.NoError(t, err)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a, resolveErr := dicontext.Resolve[testtypes.InterfaceA](r.Context())
			assert.NotNil(t, a)
			assert.NoError(t, resolveErr)

			w.WriteHeader(http.StatusOK)
		})

		code := RunRequest(t, mw(handler))
		assert.Equal(t, http.StatusOK, code)

		assert.True(t, called)
	})
}

func RunRequest(t *testing.T, h http.Handler) int {
	res := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/", http.NoBody)
	require.NoError(t, err)

	h.ServeHTTP(res, req)
	return res.Code
}
