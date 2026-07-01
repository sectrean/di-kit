package service

import (
	"context"
	"log/slog"
)

func NewService(logger *slog.Logger) *Service {
	logger.Info("NewService called")

	return &Service{
		logger: logger,
	}
}

type Service struct {
	logger *slog.Logger
}

func (s *Service) Run(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Service.Run called")
	return nil
}

func (s *Service) Close(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Service.Close called")
	return nil
}
