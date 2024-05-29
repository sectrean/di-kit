package bar

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/johnrutherford/di-kit/examples/foo"
)

type BarService struct {
	logger *slog.Logger
}

func NewBarService(logger *slog.Logger, _ *foo.FooService) *BarService {
	return &BarService{
		logger: logger,
	}
}

func (s *BarService) HandleRequest(r *http.Request, w http.ResponseWriter) {
	s.logger.Info("BarService.HandleRequest called")
}

func (s *BarService) Close(_ context.Context) {
	s.logger.Info("BarService.Close called")
}
