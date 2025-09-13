package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/Blazingkevin/loki/internal/openapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
	spec     *openapi3.T
	specInfo *openapi.SpecInfo
	logger   *slog.Logger
}

func (s *ServerTestSuite) SetupSuite() {
	parser := openapi.NewParser()
	spec, err := parser.LoadSpec("../../test/petstore.yaml")

	require.NoError(s.T(), err, "Failed to load spec")

	s.spec = spec
	s.specInfo = openapi.AnalyzeSpec(spec)

	s.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func (s *ServerTestSuite) TestNewServer() {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "config can not be nil",
		},
		{
			name: "nil spec",
			config: &Config{
				Host: "localhost",
				Port: 8080,
				Spec: nil,
			},
			expectError: true,
			errorMsg:    "OpenAPI spec cannot be nil",
		},
		{
			name: "nil spec info",
			config: &Config{
				Host:     "localhost",
				Port:     8080,
				Spec:     s.spec,
				SpecInfo: nil,
			},
			expectError: true,
			errorMsg:    "OpenAPI spec info cannot be nil",
		},
		{
			name: "valid config without logger",
			config: &Config{
				Host:     "localhost",
				Port:     8080,
				Spec:     s.spec,
				SpecInfo: s.specInfo,
			},
			expectError: false,
		},
		{
			name: "valid config with logger",
			config: &Config{
				Host:     "localhost",
				Port:     8080,
				Spec:     s.spec,
				SpecInfo: s.specInfo,
				Logger:   s.logger,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			server, err := New(tt.config)

			if tt.expectError {
				assert.Error(s.T(), err)
				assert.Contains(s.T(), err.Error(), tt.errorMsg)
				assert.Nil(s.T(), server)
			} else {
				assert.NoError(s.T(), err)
				assert.NotNil(s.T(), server)
				assert.Equal(s.T(), server.config, tt.config)
				assert.NotNil(s.T(), server.logger)
				assert.NotNil(s.T(), server.logger)
			}
		})
	}

}

func (s *ServerTestSuite) TestServerStartAndShutdown() {
	config := &Config{
		Host:     "localhost",
		Port:     0,
		Spec:     s.spec,
		SpecInfo: s.specInfo,
		Logger:   s.logger,
	}

	server, err := New(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://" + server.Addr() + "/_loki/health")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func (s *ServerTestSuite) TestServerHealthEndpoint() {
	config := &Config{
		Host:     "localhost",
		Port:     0,
		Spec:     s.spec,
		SpecInfo: s.specInfo,
		Logger:   s.logger,
	}

	server, err := New(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://" + server.Addr() + "/_loki/health")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(s.T(), "application/json", resp.Header.Get("Content-Type"))

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "healthy", health["status"])
	assert.Equal(s.T(), "loki-mock-server", health["service"])

	specInfo, ok := health["spec_info"].(map[string]interface{})
	require.True(s.T(), ok)
	assert.Equal(s.T(), "Sample Pet Store API", specInfo["title"])
	assert.Equal(s.T(), "1.0.0", specInfo["version"])
}

func (s *ServerTestSuite) TestServerSpecInfoEndpoint() {
	config := &Config{
		Host:     "localhost",
		Port:     0,
		Spec:     s.spec,
		SpecInfo: s.specInfo,
		Logger:   s.logger,
	}

	server, err := New(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://" + server.Addr() + "/_loki/spec")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(s.T(), "application/json", resp.Header.Get("Content-Type"))

	var specInfo openapi.SpecInfo
	err = json.NewDecoder(resp.Body).Decode(&specInfo)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "Sample Pet Store API", specInfo.Title)
	assert.Equal(s.T(), "1.0.0", specInfo.Version)
	assert.Equal(s.T(), 2, specInfo.PathCount)
	assert.Len(s.T(), specInfo.Paths, 2)
}
