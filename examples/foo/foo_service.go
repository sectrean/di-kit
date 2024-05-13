package foo

import (
	"context"
	"log/slog"
)

func NewFooService(logger *slog.Logger) *FooService {
	logger.Info("NewFooService called")

	return &FooService{
		logger: logger,
	}
}

type FooService struct {
	logger *slog.Logger
}

func (s *FooService) Run(ctx context.Context) error {
	s.logger.InfoContext(ctx, "FooService.Run called")
	return nil
}

func (s *FooService) Close(ctx context.Context) error {
	s.logger.InfoContext(ctx, "FooService.Close called")
	return nil
}
