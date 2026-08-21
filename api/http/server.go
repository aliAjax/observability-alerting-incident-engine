package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"golang.org/x/time/rate"

	"github.com/observability-alerting/engine/internal/config"
)

type Server struct {
	server *http.Server
	logger *slog.Logger
}

func NewServer(cfg config.Config, deps Dependencies, logger *slog.Logger) *Server {
	limiter := rate.NewLimiter(rate.Limit(cfg.HTTP.RateLimitPerSec), cfg.HTTP.Burst)
	handler := Middleware(logger, limiter, cfg.HTTP.WriteTimeout)(deps.Routes())
	return &Server{
		server: &http.Server{
			Addr:         cfg.HTTP.Address,
			Handler:      handler,
			ReadTimeout:  cfg.HTTP.ReadTimeout,
			WriteTimeout: cfg.HTTP.WriteTimeout,
		},
		logger: logger,
	}
}

func (s *Server) Start() error {
	s.logger.Info("starting http server", "address", s.server.Addr)
	err := s.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down http server")
	return s.server.Shutdown(ctx)
}
