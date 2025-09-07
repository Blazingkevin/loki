package openapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestNewParser(t *testing.T) {
	parser := NewParser()

	if parser == nil {
		t.Fatal("NewParser() returned nil")
	}

	if parser.client == nil {
		t.Fatal("Parser client is nil")
	}

	if parser.client.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout of 30s but got %v", parser.client.Timeout)
	}

	customTimeout := 10 * time.Second
	parser2 := NewParser(WithTimeout(customTimeout))
	if parser2.client.Timeout != customTimeout {
		t.Errorf("Expected timeout of %v but got %v", customTimeout, parser2.client.Timeout)
	}

	customClient := &http.Client{Timeout: 5 * time.Second}
	parser3 := NewParser(WithHttpClient(customClient))
	if parser3.client.Timeout != customClient.Timeout {
		t.Errorf("Expected timeout of %v but got %v", customClient.Timeout, parser3.client.Timeout)
	}
}

func TestLoadSpecFromFile(t *testing.T) {
	parser := NewParser()

	spec, err := parser.LoadSpec("../../test/petstore.yaml")

	if err != nil {
		t.Errorf("Failed to load valid spec: %v", err)
	}

	if spec == nil {
		t.Fatal("LoadSpec returned nil spec")
	}

	if spec.Info == nil {
		t.Error("Spec Info is nil")
	}

	if spec.Info.Title != "Sample Pet Store API" {
		t.Errorf("Expected title 'Sample Pet Store API', got %q", spec.Info.Title)
	}

	if spec.Info.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %q", spec.Info.Version)
	}

	if spec.Paths == nil || spec.Paths.Len() == 0 {
		t.Error("No paths found in spec")
	}

	expectedPaths := []string{"/pets", "/pets/{petId}"}
	for _, path := range expectedPaths {
		if spec.Paths.Value(path) == nil {
			t.Errorf("Expected path %q not found", path)
		}
	}
}

func TestLoadSpecFromNonExistentFile(t *testing.T) {
	parser := NewParser()

	_, err := parser.LoadSpec("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	if !strings.Contains(err.Error(), "failed to open file") {
		t.Errorf("Expected 'failed to open file' error, got: %v", err)
	}
}

func TestLoadSpecFromInvalidFile(t *testing.T) {
	parser := NewParser()

	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(invalidFile, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = parser.LoadSpec(invalidFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}

	if !strings.Contains(err.Error(), "failed to parse OpenAPI spec") {
		t.Errorf("Expected 'failed to parse OpenAPI spec' error, got: %v", err)
	}
}

func TestLoadSpecFromURL(t *testing.T) {
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
		if userAgent := r.Header.Get("User-Agent"); !strings.Contains(userAgent, "Loki-API-Mocker") {
			t.Errorf("Expected User-Agent to contain 'Loki-API-Mocker', got: %q", userAgent)
		}

		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validSpec))
	}))
	defer server.Close()

	parser := NewParser()
	spec, err := parser.LoadSpec(server.URL)
	if err != nil {
		t.Fatalf("Failed to load spec from URL: %v", err)
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got %q", spec.Info.Title)
	}
}

func TestLoadSpecFromURLWithError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	parser := NewParser()
	_, err := parser.LoadSpec(server.URL)
	if err == nil {
		t.Error("Expected error for 404 response")
	}

	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("Expected 'HTTP 404' error, got: %v", err)
	}
}

func TestLoadSpecFromInvalidURL(t *testing.T) {
	parser := NewParser()

	_, err := parser.LoadSpec("http://nonexistent/spec.yaml")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestValidateSpec(t *testing.T) {
	parser := NewParser()

	spec, err := parser.LoadSpec("../../test/petstore.yaml")
	if err != nil {
		t.Fatalf("Failed to load spec: %v", err)
	}

	err = parser.ValidateSpec(spec)
	if err != nil {
		t.Errorf("Valid spec failed validation: %v", err)
	}
}

func TestValidateInvalidSpec(t *testing.T) {
	parser := NewParser()

	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Paths:   openapi3.NewPaths(),
	}

	err := parser.ValidateSpec(spec)
	if err == nil {
		t.Error("Expected validation error for invalid spec")
	}
}

func TestLoadAndValidate(t *testing.T) {
	parser := NewParser()

	spec, err := parser.LoadAndValidate("../../test/petstore.yaml")
	if err != nil {
		t.Fatalf("LoadAndValidate failed: %v", err)
	}

	if spec == nil {
		t.Fatal("LoadAndValidate returned nil spec")
	}

	_, err = parser.LoadAndValidate("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"http://example.com", true},
		{"https://api.example.com", true},
		{"https://api.example.com/spec.yaml", true},
		{"ftp://example.com", false},
		{"file.yaml", false},
		{"./file.yaml", false},
		{"/path/to/file.yaml", false},
		{"", false},
		{"not-a-url", false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := isURL(test.input)
			if result != test.expected {
				t.Errorf("isURL(%q) = %v, expected %v", test.input, result, test.expected)
			}
		})
	}
}

func BenchmarkLoadSpecFromFile(b *testing.B) {
	parser := NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.LoadSpec("../../test/petstore.yaml")
		if err != nil {
			b.Fatalf("LoadSpec failed: %v", err)
		}
	}
}

func BenchmarkValidateSpec(b *testing.B) {
	parser := NewParser()
	spec, err := parser.LoadSpec("../../test/petstore.yaml")
	if err != nil {
		b.Fatalf("Failed to load spec: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := parser.ValidateSpec(spec)
		if err != nil {
			b.Fatalf("ValidateSpec failed: %v", err)
		}
	}
}
