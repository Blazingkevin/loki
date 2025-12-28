package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseWriter(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := newResponseWriter(w)

		rw.WriteHeader(http.StatusNotFound)
		assert.Equal(t, http.StatusNotFound, rw.statusCode)
	})

	t.Run("defaults to 200 OK", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := newResponseWriter(w)

		rw.Write([]byte("test"))
		assert.Equal(t, http.StatusOK, rw.statusCode)
	})

	t.Run("prevents multiple WriteHeader calls", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := newResponseWriter(w)

		rw.WriteHeader(http.StatusNotFound)
		rw.WriteHeader(http.StatusOK) // Should be ignored

		assert.Equal(t, http.StatusNotFound, rw.statusCode)
	})
}

func TestLoggingMiddleware(t *testing.T) {
	t.Run("adds request ID if not present", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLogger("info", buf)

		handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID)
	})

	t.Run("preserves existing request ID", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLogger("info", buf)

		handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("X-Request-ID", "custom-id-123")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		assert.Equal(t, "custom-id-123", requestID)
	})

	t.Run("logs response with status code and duration", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLogger("info", buf)

		handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))

		req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		output := buf.String()
		assert.Contains(t, output, "response sent")
		assert.Contains(t, output, "POST")
		assert.Contains(t, output, "/api/users")
		assert.Contains(t, output, "201")
		assert.Contains(t, output, "duration_ms")
	})

	t.Run("logs request in debug mode", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLogger("debug", buf)

		handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		output := buf.String()
		assert.Contains(t, output, "incoming request")
		assert.Contains(t, output, "application/json")
	})

	t.Run("does not log request details in info mode", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLogger("info", buf)

		handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		output := buf.String()
		assert.NotContains(t, output, "incoming request")
		assert.Contains(t, output, "response sent")
	})
}

func TestCORSMiddleware(t *testing.T) {
	t.Run("adds CORS headers", func(t *testing.T) {
		handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		assert.Contains(t, w.Header().Get("Access-Control-Expose-Headers"), "X-Request-ID")
	})

	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		handlerCalled := false
		handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.False(t, handlerCalled, "Handler should not be called for OPTIONS")
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLogger("error", buf)

		handler := RecoveryMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("X-Request-ID", "panic-test-123")
		w := httptest.NewRecorder()

		// Should not panic
		require.NotPanics(t, func() {
			handler.ServeHTTP(w, req)
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Internal server error")

		output := buf.String()
		assert.Contains(t, output, "panic recovered")
		assert.Contains(t, output, "test panic")
		assert.Contains(t, output, "panic-test-123")
	})

	t.Run("does not interfere with normal requests", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLogger("error", buf)

		handler := RecoveryMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())
		assert.Empty(t, buf.String())
	})
}

func TestExtractHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("X-Request-ID", "req-123")

	headers := extractHeaders(req)

	assert.Contains(t, headers, "Content-Type")
	assert.Contains(t, headers, "User-Agent")
	assert.Contains(t, headers, "X-Request-ID")
	assert.NotContains(t, headers, "Authorization", "Should not log sensitive headers")
}

func TestMiddlewareChaining(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("info", buf)

	// Chain all middleware together
	handler := RecoveryMiddleware(logger)(
		CORSMiddleware(
			LoggingMiddleware(logger)(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status":"ok"}`))
				}),
			),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"status":"ok"}`, w.Body.String())

	// Verify CORS headers
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// Verify request ID
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))

	// Verify logging
	output := buf.String()
	assert.Contains(t, output, "response sent")
	assert.Contains(t, output, "/api/health")
}
