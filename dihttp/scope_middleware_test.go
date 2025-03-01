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

	t.Run("multiple middleware calls", func(t *testing.T) {
		c, err := di.NewContainer()
		require.NoError(t, err)

		mw, err := dihttp.NewRequestScopeMiddleware(c)
		require.NoError(t, err)

		handlerA := mw(http.NotFoundHandler())
		handlerB := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(500)
		}))

		gotA := RunRequest(t, handlerA, "/")
		assert.Equal(t, http.StatusNotFound, gotA)

		gotB := RunRequest(t, handlerB, "/")
		assert.Equal(t, http.StatusInternalServerError, gotB)
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

		code := RunRequest(t, mw(handler), "/")
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

		code := RunRequest(t, mw(handler), "/")
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

		code := RunRequest(t, mw(handler), "/")
		assert.Equal(t, http.StatusOK, code)
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

		mw, err := dihttp.NewRequestScopeMiddleware(c)
		require.NoError(t, err)

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

		code := RunRequest(t, mw(handler), "/")
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

		code := RunRequest(t, mw(handler), "/")
		assert.Equal(t, http.StatusOK, code)

		assert.True(t, called)
	})
}

func RunRequest(t *testing.T, h http.Handler, path string) int {
	res := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, path, http.NoBody)
	require.NoError(t, err)

	h.ServeHTTP(res, req)
	return res.Code
}
