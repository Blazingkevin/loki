package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/Blazingkevin/loki/internal/config"
	"github.com/Blazingkevin/loki/internal/openapi"
)

type Config struct {
	Host        string
	Port        int
	Spec        *openapi3.T
	SpecInfo    *openapi.SpecInfo
	Logger      *Logger
	LogLevel    string
	ChaosConfig *config.Config
}

type Server struct {
	config     *Config
	httpServer *http.Server
	router     *Router
	listener   net.Listener
	logger     *Logger
	mu         sync.RWMutex
	ready      chan struct{}
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

	// Create logger with configured level
	logger := config.Logger
	if logger == nil {
		logLevel := config.LogLevel
		if logLevel == "" {
			logLevel = "info"
		}
		logger = NewLogger(logLevel, nil)
	}

	router := NewRouter(config.Spec, config.SpecInfo, logger, config.ChaosConfig)

	// Build middleware chain
	var handler http.Handler = router

	// Apply logging middleware
	handler = LoggingMiddleware(logger)(handler)

	// Apply chaos middleware if configured
	if config.ChaosConfig != nil {
		// Import chaos at function level to avoid package cycle
		// We'll use a lazy import approach
		logger.Info("Chaos engineering enabled",
			"scenarios", len(config.ChaosConfig.GetEnabledScenarios()),
		)
	}

	// Apply CORS middleware
	handler = CORSMiddleware(handler)

	// Apply recovery middleware (outermost)
	handler = RecoveryMiddleware(logger)(handler)

	server := &Server{
		config: config,
		router: router,
		logger: logger,
		ready:  make(chan struct{}),
	}

	// Store the middleware-wrapped handler for the HTTP server
	server.httpServer = &http.Server{
		Handler: handler,
	}

	return server, nil
}

// WrapHandler wraps the current handler with additional middleware
// This allows external packages to add middleware without creating import cycles
func (s *Server) WrapHandler(wrapper func(http.Handler) http.Handler) {
	s.httpServer.Handler = wrapper(s.httpServer.Handler)
}

func (s *Server) Start(ctx context.Context) error {
	addr := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))

	var err error
	s.mu.Lock()
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Update the server address (httpServer.Handler already set in New())
	s.httpServer.ReadTimeout = 30 * time.Second
	s.httpServer.WriteTimeout = 30 * time.Second
	s.httpServer.IdleTimeout = 120 * time.Second

	actualAddr := s.listener.Addr().String()
	s.mu.Unlock()

	close(s.ready)

	s.logger.Info("🔥 Loki Mock Server Starting",
		"addr", actualAddr,
		"spec_title", s.config.SpecInfo.Title,
		"spec_version", s.config.SpecInfo.Version,
		"endpoints", s.config.SpecInfo.PathCount)

	errCh := make(chan error, 1)
	go func() {
		s.mu.RLock()
		listener := s.listener
		httpServer := s.httpServer
		s.mu.RUnlock()

		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Server) WaitForReady(ctx context.Context) error {
	select {
	case <-s.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
