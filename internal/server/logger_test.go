package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected string
	}{
		{"debug level", "debug", "debug"},
		{"info level", "info", "info"},
		{"warn level", "warn", "warn"},
		{"warning level", "warning", "warn"},
		{"error level", "error", "error"},
		{"invalid level defaults to info", "invalid", "info"},
		{"empty level defaults to info", "", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewLogger(tt.level, buf)
			assert.NotNil(t, logger)
			assert.NotNil(t, logger.Logger)
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "DEBUG"},
		{"DEBUG", "DEBUG"},
		{"info", "INFO"},
		{"INFO", "INFO"},
		{"warn", "WARN"},
		{"warning", "WARN"},
		{"error", "ERROR"},
		{"invalid", "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := parseLogLevel(tt.input)
			assert.Equal(t, tt.expected, level.String())
		})
	}
}

func TestLoggerWithRequestID(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("info", buf)

	requestLogger := logger.WithRequestID("test-request-id")
	requestLogger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "test-request-id")
	assert.Contains(t, output, "test message")
}

func TestLoggerWithEndpoint(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("info", buf)

	endpointLogger := logger.WithEndpoint("GET", "/api/users")
	endpointLogger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/api/users")
	assert.Contains(t, output, "test message")
}

func TestLogRequest(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("info", buf)

	headers := map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   "test-agent",
	}

	logger.LogRequest("POST", "/api/users", "req-123", headers)

	output := buf.String()
	assert.Contains(t, output, "incoming request")
	assert.Contains(t, output, "POST")
	assert.Contains(t, output, "/api/users")
	assert.Contains(t, output, "req-123")
	assert.Contains(t, output, "application/json")
}

func TestLogResponse(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("info", buf)

	logger.LogResponse("GET", "/api/users", "req-123", 200, 45.5)

	output := buf.String()
	assert.Contains(t, output, "response sent")
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/api/users")
	assert.Contains(t, output, "req-123")
	assert.Contains(t, output, "200")
	assert.Contains(t, output, "45.5")
}

func TestLogChaosApplied(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("info", buf)

	details := map[string]interface{}{
		"latency_ms":   100,
		"distribution": "uniform",
	}

	logger.LogChaosApplied("req-123", "latency_scenario", "latency", details)

	output := buf.String()
	assert.Contains(t, output, "chaos applied")
	assert.Contains(t, output, "req-123")
	assert.Contains(t, output, "latency_scenario")
	assert.Contains(t, output, "latency")
}

func TestLogError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("error", buf)

	err := errors.New("test error")
	logger.LogError(err, "req-123", "database connection")

	output := buf.String()
	assert.Contains(t, output, "error occurred")
	assert.Contains(t, output, "test error")
	assert.Contains(t, output, "req-123")
	assert.Contains(t, output, "database connection")
}

func TestIsDebug(t *testing.T) {
	tests := []struct {
		level    string
		expected bool
	}{
		{"debug", true},
		{"info", false},
		{"warn", false},
		{"error", false},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewLogger(tt.level, buf)
			assert.Equal(t, tt.expected, logger.IsDebug())
		})
	}
}

func TestLoggerJSONOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("info", buf)

	logger.Info("test message", "key", "value")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "INFO", logEntry["level"])
	assert.Equal(t, "test message", logEntry["msg"])
	assert.Equal(t, "value", logEntry["key"])
	assert.NotEmpty(t, logEntry["time"])
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		logFunc   func(*Logger)
		shouldLog bool
	}{
		{
			name:     "debug logs at debug level",
			logLevel: "debug",
			logFunc: func(l *Logger) {
				l.Debug("debug message")
			},
			shouldLog: true,
		},
		{
			name:     "debug does not log at info level",
			logLevel: "info",
			logFunc: func(l *Logger) {
				l.Debug("debug message")
			},
			shouldLog: false,
		},
		{
			name:     "info logs at info level",
			logLevel: "info",
			logFunc: func(l *Logger) {
				l.Info("info message")
			},
			shouldLog: true,
		},
		{
			name:     "warn logs at warn level",
			logLevel: "warn",
			logFunc: func(l *Logger) {
				l.Warn("warn message")
			},
			shouldLog: true,
		},
		{
			name:     "error logs at error level",
			logLevel: "error",
			logFunc: func(l *Logger) {
				l.Error("error message")
			},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewLogger(tt.logLevel, buf)

			tt.logFunc(logger)

			output := buf.String()
			if tt.shouldLog {
				assert.NotEmpty(t, output)
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

func TestPretty(t *testing.T) {
	result := Pretty("test %s with %d items", "message", 5)
	assert.Equal(t, "test message with 5 items", result)
}
