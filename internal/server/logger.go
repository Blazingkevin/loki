package server

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Logger provides structured logging capabilities
type Logger struct {
	*slog.Logger
	level slog.Level
}

// NewLogger creates a new logger with the specified level
func NewLogger(level string, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}

	logLevel := parseLogLevel(level)

	opts := &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "time",
					Value: a.Value,
				}
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(output, opts)
	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
		level:  logLevel,
	}
}

// parseLogLevel converts string level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID returns a logger with a request ID field
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.With(slog.String("request_id", requestID)),
		level:  l.level,
	}
}

// WithEndpoint returns a logger with endpoint information
func (l *Logger) WithEndpoint(method, path string) *Logger {
	return &Logger{
		Logger: l.With(
			slog.String("method", method),
			slog.String("path", path),
		),
		level: l.level,
	}
}

// LogRequest logs an incoming HTTP request
func (l *Logger) LogRequest(method, path, requestID string, headers map[string]string) {
	l.Info("incoming request",
		slog.String("method", method),
		slog.String("path", path),
		slog.String("request_id", requestID),
		slog.Any("headers", headers),
	)
}

// LogResponse logs an outgoing HTTP response
func (l *Logger) LogResponse(method, path, requestID string, statusCode int, duration float64) {
	l.Info("response sent",
		slog.String("method", method),
		slog.String("path", path),
		slog.String("request_id", requestID),
		slog.Int("status_code", statusCode),
		slog.Float64("duration_ms", duration),
	)
}

// LogChaosApplied logs when chaos engineering is applied to a request
func (l *Logger) LogChaosApplied(requestID, scenario, chaosType string, details map[string]interface{}) {
	l.Info("chaos applied",
		slog.String("request_id", requestID),
		slog.String("scenario", scenario),
		slog.String("chaos_type", chaosType),
		slog.Any("details", details),
	)
}

// LogError logs an error with context
func (l *Logger) LogError(err error, requestID, context string) {
	l.Error("error occurred",
		slog.String("error", err.Error()),
		slog.String("request_id", requestID),
		slog.String("context", context),
	)
}

// IsDebug returns true if debug logging is enabled
func (l *Logger) IsDebug() bool {
	return l.level <= slog.LevelDebug
}

// Pretty formats a message for console output (non-JSON)
func Pretty(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
