package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/Blazingkevin/loki/internal/openapi"
)

type Router struct {
	spec     *openapi3.T
	specInfo *openapi.SpecInfo
	logger   *Logger
	mux      *http.ServeMux
}

func NewRouter(spec *openapi3.T, specInfo *openapi.SpecInfo, logger *Logger) *Router {
	router := &Router{
		spec:     spec,
		specInfo: specInfo,
		logger:   logger,
		mux:      http.NewServeMux(),
	}

	router.registerRoutes()

	return router
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Middleware chain is applied in server.go, not here
	// This method now just delegates to the mux

	r.mux.ServeHTTP(w, req)
}

func (r *Router) registerRoutes() {
	r.mux.HandleFunc("/_loki/health", r.handleHealth)
	r.mux.HandleFunc("/_loki/spec", r.handleSpecInfo)

	for _, pathInfo := range r.specInfo.Paths {
		// options handler for CORS
		pattern := pathInfo.Path
		optionsHandler := func(w http.ResponseWriter, req *http.Request) {
			if req.Method == "OPTIONS" {
				r.setCORSHeaders(w)
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		r.mux.HandleFunc("OPTIONS "+pattern, optionsHandler)

		for _, methodInfo := range pathInfo.Methods {
			r.logger.Debug("📋 Registering route",
				"method", methodInfo.Method,
				"path", pathInfo.Path,
				"pattern", pattern,
				"operation_id", methodInfo.OperationID)

			handler := r.createHandler(pathInfo, methodInfo)

			// handler for specific method and path
			// I should consider a better implementation that is more compatible with Go versions prior to 1.22
			methodPattern := fmt.Sprintf("%s %s", methodInfo.Method, pattern)
			r.mux.HandleFunc(methodPattern, handler)
		}
	}
}

func (r *Router) createHandler(pathInfo openapi.PathInfo, methodInfo openapi.MethodInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		r.handleAPIEndpoint(w, req, pathInfo, methodInfo)
	}
}

func (r *Router) handleAPIEndpoint(w http.ResponseWriter, req *http.Request, pathInfo openapi.PathInfo, methodInfo openapi.MethodInfo) {
	r.logger.Info("🎯 Handling API endpoint",
		"method", methodInfo.Method,
		"path", pathInfo.Path,
		"operation_id", methodInfo.OperationID)

	r.setCORSHeaders(w)

	if req.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// TODO: Implement proper response generation from the OpenAPI schemas
	response := map[string]interface{}{
		"message":      "Loki mock response",
		"method":       methodInfo.Method,
		"path":         pathInfo.Path,
		"operation_id": methodInfo.OperationID,
		"timestamp":    "2024-09-07T12:00:00Z",
	}

	statusCode := r.getDefaultStatusCode(methodInfo)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		r.logger.Error("❌ Failed to encode response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	r.logger.Info("📤 Response sent",
		"status_code", statusCode,
		"content_type", "application/json")
}

func (r *Router) getDefaultStatusCode(methodInfo openapi.MethodInfo) int {
	for _, code := range methodInfo.Responses {
		switch code {
		case "200":
			return http.StatusOK
		case "201":
			return http.StatusCreated
		case "202":
			return http.StatusAccepted
		case "204":
			return http.StatusNoContent
		}
	}

	switch methodInfo.Method {
	case "POST":
		return http.StatusCreated
	case "PUT", "PATCH":
		return http.StatusOK
	case "DELETE":
		return http.StatusNoContent
	default:
		return http.StatusOK
	}
}

func (r *Router) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	w.Header().Set("Access-Control-Max-Age", "3600")
}

func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":  "healthy",
		"service": "loki-mock-server",
		"version": "1.0.0", // TODO: need to get thins from build info
		"spec_info": map[string]interface{}{
			"title":       r.specInfo.Title,
			"version":     r.specInfo.Version,
			"endpoints":   r.specInfo.PathCount,
			"server_urls": r.specInfo.ServerURLs,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(health) //nolint:errcheck // Response encoding errors are not critical for health check
}

func (r *Router) handleSpecInfo(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(r.specInfo) //nolint:errcheck // Response encoding errors are not critical for spec info endpoint
}

func (r *Router) GetRegisteredRoutes() []string {
	routes := []string{
		"GET /_loki/health",
		"GET /_loki/spec",
	}

	for _, pathInfo := range r.specInfo.Paths {
		for _, methodInfo := range pathInfo.Methods {
			route := fmt.Sprintf("%s %s", methodInfo.Method, pathInfo.Path)
			routes = append(routes, route)
		}
	}

	return routes
}
