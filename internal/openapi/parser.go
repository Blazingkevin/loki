package openapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

type Parser struct {
	client *http.Client
}

type ParserOption func(*Parser)

func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func WithHTTPClient(client *http.Client) ParserOption {
	return func(p *Parser) {
		p.client = client
	}
}

func WithTimeout(timeout time.Duration) ParserOption {
	return func(p *Parser) {
		p.client.Timeout = timeout
	}
}

func (p *Parser) LoadSpec(source string) (*openapi3.T, error) {
	if isURL(source) {
		return p.LoadFromURL(source)
	}

	return p.LoadFromFile(source)
}

func (p *Parser) LoadFromURL(specUrl string) (*openapi3.T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, specUrl, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request %q: %w", specUrl, err)
	}

	req.Header.Set("Accept", "application/json, application/yaml, text/yaml")
	req.Header.Set("User-Agent", "Loki-API-Mocker")

	resp, err := p.client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch spec %q: %w", specUrl, err)
	}
	defer resp.Body.Close() //nolint:errcheck // Body will be read before close

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch spec from %q: HTTP %d", specUrl, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %q : %w", specUrl, err)
	}

	return p.ParseSpec(data, specUrl)
}

func (p *Parser) LoadFromFile(source string) (*openapi3.T, error) {
	file, err := os.Open(source)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", source, err)
	}
	defer file.Close() //nolint:errcheck // File will be read before close

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", source, err)
	}

	return p.ParseSpec(data, source)
}

func (p *Parser) ParseSpec(data []byte, source string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	spec, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec from %q: %w", source, err)
	}

	return spec, nil
}

func (p *Parser) ValidateSpec(spec *openapi3.T) error {
	ctx := context.Background()
	return spec.Validate(ctx)
}

func (p *Parser) LoadAndValidate(source string) (*openapi3.T, error) {
	spec, err := p.LoadSpec(source)
	if err != nil {
		return nil, err
	}

	if err := p.ValidateSpec(spec); err != nil {
		return nil, fmt.Errorf("spec validation failed: %w", err)
	}

	return spec, nil
}

func isURL(str string) bool {
	if !strings.Contains(str, "://") {
		return false
	}

	u, err := url.Parse(str)

	if err != nil {
		return false
	}

	return u.Scheme == "http" || u.Scheme == "https"
}
