package handler

import (
	"context"
	"log/slog"
	"net/http"
)

type RequestHandler struct {
	logger *slog.Logger
	r      *http.Request
}

func NewRequestHandler(logger *slog.Logger, r *http.Request) (*RequestHandler, error) {
	return &RequestHandler{
		logger: logger,
		r:      r,
	}, nil
}

func (h *RequestHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("handling request", "method", r.Method, "url", r.URL.String())
}

func (h *RequestHandler) Close(ctx context.Context) error {
	h.logger.InfoContext(ctx, "RequestHandler.Close called")
	return nil
}
