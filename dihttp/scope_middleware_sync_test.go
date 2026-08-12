//go:build go1.25

package dihttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/dicontext"
	"github.com/sectrean/di-kit/dihttp"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ScopeMiddleware_CloseTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const closeTimeout = time.Minute

		var closed bool

		c, err := di.NewContainer(
			di.WithService(testtypes.NewInterfaceA,
				di.Scoped,
				di.UseCloseFunc(func(ctx context.Context, _ testtypes.InterfaceA) error {
					closed = true

					closeDeadline, hasDeadline := ctx.Deadline()
					assert.True(t, hasDeadline)
					assert.Equal(t, time.Now().Add(closeTimeout), closeDeadline)
					assert.NoError(t, ctx.Err())

					time.Sleep(closeTimeout - time.Nanosecond)
					synctest.Wait()
					assert.NoError(t, ctx.Err())

					time.Sleep(time.Nanosecond)
					synctest.Wait()
					assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)

					return nil
				}),
			),
		)
		require.NoError(t, err)

		mw := dihttp.NewRequestScopeMiddleware(c,
			dihttp.WithScopeCloseTimeout(closeTimeout),
		)

		reqCtx, cancel := context.WithCancel(t.Context())
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = dicontext.MustResolve[testtypes.InterfaceA](r.Context())
			cancel()
			w.WriteHeader(http.StatusOK)
		})

		res := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "/", http.NoBody)
		require.NoError(t, err)

		mw(handler).ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.True(t, closed)
	})
}
