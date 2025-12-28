package server

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// LoggingMiddleware provides request/response logging with correlation IDs
func LoggingMiddleware(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate or extract request ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Add request ID to response headers
			w.Header().Set("X-Request-ID", requestID)

			// Wrap response writer to capture status code
			rw := newResponseWriter(w)

			// Log incoming request if debug mode
			if logger.IsDebug() {
				headers := extractHeaders(r)
				logger.LogRequest(r.Method, r.URL.Path, requestID, headers)
			}

			// Track request duration
			start := time.Now()

			// Process request
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start).Seconds() * 1000 // Convert to milliseconds

			// Log response
			logger.LogResponse(r.Method, r.URL.Path, requestID, rw.statusCode, duration)
		})
	}
}

// CORSMiddleware adds CORS headers to responses
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics and logs them
func RecoveryMiddleware(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := r.Header.Get("X-Request-ID")
					if requestID == "" {
						requestID = w.Header().Get("X-Request-ID")
					}

					logger.Error("panic recovered",
						"error", err,
						"request_id", requestID,
						"method", r.Method,
						"path", r.URL.Path,
					)

					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":"Internal server error"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// extractHeaders extracts relevant headers for logging
func extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)

	// Only log specific headers to avoid logging sensitive data
	relevantHeaders := []string{
		"Content-Type",
		"User-Agent",
		"Accept",
		"X-Request-ID",
		"X-Forwarded-For",
	}

	for _, key := range relevantHeaders {
		if value := r.Header.Get(key); value != "" {
			headers[key] = value
		}
	}

	return headers
}
