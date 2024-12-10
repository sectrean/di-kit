package dihttp_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/dicontext"
	"github.com/johnrutherford/di-kit/dihttp"
	"github.com/johnrutherford/di-kit/internal/testtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RequestScopeMiddleware(t *testing.T) {
	c, err := di.NewContainer(
		di.WithService(testtypes.NewInterfaceA),
		di.WithService(func(a testtypes.InterfaceA, r *http.Request) testtypes.InterfaceB {
			assert.NotNil(t, a)
			assert.NotNil(t, r)

			return &testtypes.StructB{}
		}, di.Scoped),
	)
	require.NoError(t, err)

	var handler http.Handler
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		b, resolveErr := dicontext.Resolve[testtypes.InterfaceB](ctx)
		assert.NotNil(t, b)
		assert.NoError(t, resolveErr)

		c, resolveErr := dicontext.Resolve[testtypes.InterfaceC](ctx)
		assert.NotNil(t, c)
		assert.NoError(t, resolveErr)

		req, resolveErr := dicontext.Resolve[*http.Request](ctx)
		assert.NotNil(t, req)
		assert.NoError(t, resolveErr)

		w.WriteHeader(http.StatusOK)
	})

	middleware, err := dihttp.NewRequestScopeMiddleware(c,
		dihttp.WithContainerOptions(
			di.WithService(testtypes.NewInterfaceC),
		),
	)
	require.NoError(t, err)

	handler = middleware(handler)

	testRequest := func(t *testing.T) {
		res := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		require.NoError(t, err)

		handler.ServeHTTP(res, req)
		assert.Equal(t, http.StatusOK, res.Code)
	}

	t.Run("single request", testRequest)

	t.Run("concurrent requests", func(t *testing.T) {
		const count = 20

		wg := sync.WaitGroup{}
		wg.Add(count)

		for i := 0; i < count; i++ {
			go func() {
				testRequest(t)
				wg.Done()
			}()
		}

		wg.Wait()
	})
}
