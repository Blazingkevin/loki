package server

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"

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
	config     Config
	httpServer *http.Server
	router     *Router
	listener   net.Listener
	logger     *slog.Logger
}

func New(config *Config) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("Config can not be nil")
	}

	if config.Spec == nil {
		return nil, fmt.Errorf("Config spec can not be nil")
	}

	if config.SpecInfo == nil {
		return nil, fmt.Errorf("Config spec info can not be nil")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	router := NewRouter(config.Spec, config.SpecInfo, logger)

	server := &Server{
		config: *config,
		router: router,
		logger: logger,
	}

	return server, nil
}
