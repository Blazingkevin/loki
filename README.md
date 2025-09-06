<div align="center">
  <img src="https://raw.githubusercontent.com/Blazingkevin/loki/main/assets/loki.jpg" alt="Loki Logo" width="200" height="200">
  
  # Loki: API Mocking with Chaos Engineering
  
  *The Trickster for Your APIs* 🎭
</div>

![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)
![License](https://img.shields.io/badge/license-MIT-green)

Loki is an open-source Go library and CLI tool that generates realistic mock APIs from OpenAPI specifications while introducing configurable chaos engineering patterns. Name after the Norse trickster god, Loki helps developers test application resilience against real-world API failures and network conditions.

## Quick start


### Installation

```bash
    go install github.com/Blazingkevin/loki@latest
```

### Basic Usage

```bash
# Start a mock server from OpenAPI spec
loki serve api-spec.yaml

# Add chaos engineering
loki server api-spec.yaml --chaos chaos-config.yaml

# Validate OpenAPI specification
loki validate api-spec.yaml
```


## Development

### Prerequisite

- Go 1.21 or higher
- git

### Setup

```bash
git clone https://github.com/Blazingkevin/loki
cd loki
go mod download
```

### Testing

```bash
# Run all tests
go test ./..

# Run tests with coverage
go test -cover ./..

# Run integration tests
go test -tags=integration ./..
```

### Building

```bash
# Build for current platform
go build -o bin/loki ./cmd/loki

#Cross-compile for multiple platforms
make build-all
```

## Contributing

We welcome contributions!.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
