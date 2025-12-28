package openapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type OpenAPIParserTestSuite struct {
	suite.Suite
	parser *Parser
}

func (s *OpenAPIParserTestSuite) SetupTest() {
	s.parser = NewParser()
}

func TestOpenAPIParserTestSuite(t *testing.T) {
	suite.Run(t, new(OpenAPIParserTestSuite))
}

func (s *OpenAPIParserTestSuite) TestNewParser() {
	s.Run("Creates parser with default settings", func() {
		parser := NewParser()

		assert.NotNil(s.T(), parser)
		assert.NotNil(s.T(), parser.client)
		assert.Equal(s.T(), 30*time.Second, parser.client.Timeout)
	})

	s.Run("Creates parser with custom timeout", func() {
		customTimeout := 10 * time.Second
		parser := NewParser(WithTimeout(customTimeout))

		assert.Equal(s.T(), customTimeout, parser.client.Timeout)
	})

	s.Run("Creates parser with custom HTTP client", func() {
		customClient := &http.Client{Timeout: 5 * time.Second}
		parser := NewParser(WithHTTPClient(customClient))

		assert.Equal(s.T(), 5*time.Second, parser.client.Timeout)
	})
}

func (s *OpenAPIParserTestSuite) TestLoadSpecFromFile() {
	s.Run("Successfully loads valid OpenAPI spec", func() {
		spec, err := s.parser.LoadSpec("../../test/petstore.yaml")

		assert.NoError(s.T(), err)
		assert.NotNil(s.T(), spec)
		assert.NotNil(s.T(), spec.Info)
		assert.Equal(s.T(), "Sample Pet Store API", spec.Info.Title)
		assert.Equal(s.T(), "1.0.0", spec.Info.Version)

		assert.NotNil(s.T(), spec.Paths)
		assert.Greater(s.T(), spec.Paths.Len(), 0)

		expectedPaths := []string{"/pets", "/pets/{petId}"}
		for _, path := range expectedPaths {
			assert.NotNil(s.T(), spec.Paths.Value(path), "Expected path %q not found", path)
		}
	})
}

func (s *OpenAPIParserTestSuite) TestLoadSpecFromNonExistentFile() {
	s.Run("Returns error for non-existent file", func() {
		_, err := s.parser.LoadSpec("nonexistent.yaml")

		assert.Error(s.T(), err)
		assert.Contains(s.T(), err.Error(), "failed to open file")
	})
}

func (s *OpenAPIParserTestSuite) TestLoadSpecFromInvalidFile() {
	s.Run("Returns error for invalid YAML file", func() {
		tmpDir := s.T().TempDir()
		invalidFile := filepath.Join(tmpDir, "invalid.yaml")

		err := os.WriteFile(invalidFile, []byte("invalid: yaml: content: ["), 0o600)
		assert.NoError(s.T(), err, "Failed to create test file")

		_, err = s.parser.LoadSpec(invalidFile)
		assert.Error(s.T(), err)
		assert.Contains(s.T(), err.Error(), "failed to parse OpenAPI spec")
	})
}

func (s *OpenAPIParserTestSuite) TestLoadSpecFromURL() {
	s.Run("Successfully loads spec from URL", func() {
		validSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check that proper headers are sent
			userAgent := r.Header.Get("User-Agent")
			assert.Contains(s.T(), userAgent, "Loki-API-Mocker")

			w.Header().Set("Content-Type", "application/yaml")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(validSpec))
			assert.NoError(s.T(), err)
		}))
		defer server.Close()

		spec, err := s.parser.LoadSpec(server.URL)

		assert.NoError(s.T(), err)
		assert.NotNil(s.T(), spec)
		assert.Equal(s.T(), "Test API", spec.Info.Title)
	})

	s.Run("Returns error for HTTP 404", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, err := s.parser.LoadSpec(server.URL)

		assert.Error(s.T(), err)
		assert.Contains(s.T(), err.Error(), "HTTP 404")
	})

	s.Run("Returns error for invalid URL", func() {
		_, err := s.parser.LoadSpec("http://nonexistent-domain-12345.com/spec.yaml")

		assert.Error(s.T(), err)
	})
}

func (s *OpenAPIParserTestSuite) TestValidateSpec() {
	s.Run("Validates a correct OpenAPI spec", func() {
		spec, err := s.parser.LoadSpec("../../test/petstore.yaml")
		assert.NoError(s.T(), err)

		err = s.parser.ValidateSpec(spec)
		assert.NoError(s.T(), err)
	})

	s.Run("Returns error for invalid spec", func() {
		spec := &openapi3.T{
			OpenAPI: "3.0.0",
			Paths:   openapi3.NewPaths(),
		}

		err := s.parser.ValidateSpec(spec)
		assert.Error(s.T(), err)
	})
}

func (s *OpenAPIParserTestSuite) TestLoadAndValidate() {
	s.Run("Successfully loads and validates spec", func() {
		spec, err := s.parser.LoadAndValidate("../../test/petstore.yaml")

		assert.NoError(s.T(), err)
		assert.NotNil(s.T(), spec)
	})

	s.Run("Returns error for non-existent file", func() {
		_, err := s.parser.LoadAndValidate("nonexistent.yaml")

		assert.Error(s.T(), err)
	})
}

func (s *OpenAPIParserTestSuite) TestIsURL() {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid HTTP URL", "http://example.com", true},
		{"Valid HTTPS URL", "https://api.example.com", true},
		{"Valid HTTPS URL with path", "https://api.example.com/spec.yaml", true},
		{"FTP URL should be false", "ftp://example.com", false},
		{"Local file", "file.yaml", false},
		{"Relative path", "./file.yaml", false},
		{"Absolute path", "/path/to/file.yaml", false},
		{"Empty string", "", false},
		{"Invalid string", "not-a-url", false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := isURL(tt.input)
			assert.Equal(s.T(), tt.expected, result)
		})
	}
}

func BenchmarkLoadSpecFromFile(b *testing.B) {
	parser := NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.LoadSpec("../../test/petstore.yaml")
		assert.NoError(b, err)
	}
}

func BenchmarkValidateSpec(b *testing.B) {
	parser := NewParser()
	spec, err := parser.LoadSpec("../../test/petstore.yaml")
	assert.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := parser.ValidateSpec(spec)
		assert.NoError(b, err)
	}
}
