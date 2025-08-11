package dihttp_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
	t.Run("parent nil", func(t *testing.T) {
		assert.PanicsWithValue(t, "dihttp.NewRequestScopeMiddleware: parent is nil", func() {
			dihttp.NewRequestScopeMiddleware(nil)
		})
	})

	t.Run("WithNewScopeErrorHandler nil", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		_ = dihttp.NewRequestScopeMiddleware(c,
			dihttp.WithNewScopeErrorHandler(nil),
		)
	})

	t.Run("WithScopeCloseErrorHandler nil", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		_ = dihttp.NewRequestScopeMiddleware(c,
			dihttp.WithScopeCloseErrorHandler(nil),
		)
	})
}

func Test_Middleware(t *testing.T) {
	t.Run("Resolve scoped service", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
			di.WithService(testtypes.NewInterfaceB, di.ScopedLifetime),
		)
		require.NoError(t, err)

		mw := dihttp.NewRequestScopeMiddleware(c)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			b, resolveErr := dicontext.Resolve[testtypes.InterfaceB](ctx)
			assert.NotNil(t, b)
			assert.NoError(t, resolveErr)

			w.WriteHeader(http.StatusOK)
		})

		code := RunRequest(t, mw(handler), "/")
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("Resolve *http.Request", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		mw := dihttp.NewRequestScopeMiddleware(c)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			req, resolveErr := dicontext.Resolve[*http.Request](ctx)

			assert.Equal(t, r, req.WithContext(ctx))
			assert.NoError(t, resolveErr)

			w.WriteHeader(http.StatusOK)
		})

		code := RunRequest(t, mw(handler), "/")
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("Resolve new service on child scope", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		mw := dihttp.NewRequestScopeMiddleware(c,
			dihttp.WithContainerOptions(
				di.WithService(testtypes.NewInterfaceB),
			),
		)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			b, resolveErr := dicontext.Resolve[testtypes.InterfaceB](ctx)

			assert.NotNil(t, b)
			assert.NoError(t, resolveErr)

			w.WriteHeader(http.StatusOK)
		})

		code := RunRequest(t, mw(handler), "/")
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("middleware func reused", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		mw := dihttp.NewRequestScopeMiddleware(c)

		handlerA := mw(http.NotFoundHandler())
		handlerB := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(500)
		}))

		gotA := RunRequest(t, handlerA, "/")
		assert.Equal(t, http.StatusNotFound, gotA)

		gotB := RunRequest(t, handlerB, "/")
		assert.Equal(t, http.StatusInternalServerError, gotB)
	})

	t.Run("concurrent requests", func(t *testing.T) {
		// Run a number of concurrent requests and inject the *http.Request into
		// a scoped service. Resolve the service and check that the injected request
		// matches the request passed to the handler.
		const concurrency = 1000

		c, err := di.NewContainer(
			di.WithService(func(r *http.Request) *testtypes.StructA {
				return &testtypes.StructA{
					Tag: r.URL.Path,
				}
			}, di.ScopedLifetime),
		)
		require.NoError(t, err)

		mw := dihttp.NewRequestScopeMiddleware(c)

		tags := make(chan any, concurrency)
		expectedTags := make(chan any, concurrency)

		var handler http.Handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a, resolveErr := dicontext.Resolve[*testtypes.StructA](r.Context())
			assert.NotNil(t, a)
			assert.NoError(t, resolveErr)

			assert.Equal(t, r.URL.Path, a.Tag)
			tags <- a.Tag
		})
		handler = mw(handler)

		testutils.RunParallel(concurrency, func(i int) {
			path := fmt.Sprintf("/%d", i)
			expectedTags <- path

			RunRequest(t, handler, path)
		})

		close(tags)
		close(expectedTags)

		assert.ElementsMatch(t, testutils.CollectChannel(expectedTags), testutils.CollectChannel(tags))
	})

	t.Run("NewScope error", func(t *testing.T) {
		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA),
		)
		require.NoError(t, err)

		called := false

		mw := dihttp.NewRequestScopeMiddleware(c,
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

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Fail(t, "handler should not get called")
		})

		code := RunRequest(t, mw(handler), "/")
		assert.Equal(t, 599, code)

		assert.True(t, called)
	})

	t.Run("NewScope error default handler", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		mw := dihttp.NewRequestScopeMiddleware(c,
			dihttp.WithContainerOptions(
				di.WithService(nil),
			),
		)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Fail(t, "handler should not get called")
		})

		code := RunRequest(t, mw(handler), "/")
		assert.Equal(t, 500, code)
		// TODO: Assert log output
	})

	t.Run("Close error", func(t *testing.T) {
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

		mw := dihttp.NewRequestScopeMiddleware(c,
			dihttp.WithScopeCloseErrorHandler(func(r *http.Request, err error) {
				assert.NotNil(t, r)
				assert.EqualError(t, err, "di.Container.Close: close error")
				called = true
			}),
		)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = dicontext.MustResolve[testtypes.InterfaceA](r.Context())
			w.WriteHeader(http.StatusOK)
		})

		code := RunRequest(t, mw(handler), "/")
		assert.Equal(t, http.StatusOK, code)

		assert.True(t, called)
	})

	t.Run("Close error default handler", func(t *testing.T) {
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

		mw := dihttp.NewRequestScopeMiddleware(c)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = dicontext.MustResolve[testtypes.InterfaceA](r.Context())
			w.WriteHeader(http.StatusOK)
		})

		code := RunRequest(t, mw(handler), "/")
		assert.Equal(t, http.StatusOK, code)
		// TODO: Assert log output
	})
}

func RunRequest(t *testing.T, h http.Handler, path string) int {
	res := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, path, http.NoBody)
	require.NoError(t, err)

	h.ServeHTTP(res, req)
	return res.Code
}
