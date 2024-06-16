package dihttp_test

import (
	"net/http"
	"net/http/httptest"
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

	middleware := dihttp.RequestScopeMiddleware(c)

	var handler http.Handler
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := dicontext.Resolve[testtypes.InterfaceB](r.Context())
		assert.NotNil(t, b)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	})
	handler = middleware(handler)

	res := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	handler.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
}
