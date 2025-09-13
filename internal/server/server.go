package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/Blazingkevin/loki/internal/openapi"
	"github.com/getkin/kin-openapi/openapi3"
)

type Config struct {
	Host     string
	Port     int
	Spec     *openapi3.T
	SpecInfo *openapi.SpecInfo
	Logger   *slog.Logger
}

type Server struct {
	config     *Config
	httpServer *http.Server
	router     *Router
	listener   net.Listener
	logger     *slog.Logger
}

func New(config *Config) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("config can not be nil")
	}

	if config.Spec == nil {
		return nil, fmt.Errorf("OpenAPI spec cannot be nil")
	}

	if config.SpecInfo == nil {
		return nil, fmt.Errorf("OpenAPI spec info cannot be nil")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	router := NewRouter(config.Spec, config.SpecInfo, logger)

	server := &Server{
		config: config,
		router: router,
		logger: logger,
	}

	return server, nil
}

func (s *Server) Start(ctx context.Context) error {
	addr := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))

	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	s.httpServer = &http.Server{
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	actualAddr := s.listener.Addr().String()

	s.logger.Info("🔥 Loki Mock Server Starting",
		"addr", actualAddr,
		"spec_title", s.config.SpecInfo.Title,
		"spec_version", s.config.SpecInfo.Version,
		"endpoints", s.config.SpecInfo.PathCount)

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server failed to start: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.Shutdown()
	}

}

func (s *Server) Shutdown() error {
	if s.httpServer == nil {
		return nil
	}

	s.logger.Info("🛑 Shutting down loki mock server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info("✅ server shutdown complete")
	return nil
}

func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}
