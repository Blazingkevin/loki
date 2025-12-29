package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/Blazingkevin/loki/internal/config"
	"github.com/Blazingkevin/loki/internal/generator"
	"github.com/Blazingkevin/loki/internal/openapi"
)

type Router struct {
	spec      *openapi3.T
	specInfo  *openapi.SpecInfo
	logger    *Logger
	mux       *http.ServeMux
	generator *generator.Generator
}

func NewRouter(spec *openapi3.T, specInfo *openapi.SpecInfo, logger *Logger, chaosConfig interface{}) *Router {
	gen := generator.NewGenerator()

	// Configure generator with chaos config if available
	if cfg, ok := chaosConfig.(*config.Config); ok && cfg != nil {
		if cfg.FieldMapping != nil {
			gen.SetFieldMapping(cfg.FieldMapping)
		}
		if cfg.TypeMapping != nil {
			gen.SetTypeMapping(cfg.TypeMapping)
		}
	}

	router := &Router{
		spec:      spec,
		specInfo:  specInfo,
		logger:    logger,
		mux:       http.NewServeMux(),
		generator: gen,
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

	// Get the operation from the spec
	operation := r.getOperation(pathInfo.Path, methodInfo.Method)
	statusCode := r.getDefaultStatusCode(methodInfo)

	// Try to generate response from schema
	var response interface{}
	if operation != nil {
		schema := r.getResponseSchema(operation, statusCode)
		if schema != nil {
			// Wrap schema in SchemaRef
			schemaRef := &openapi3.SchemaRef{Value: schema}
			generatedResp, err := r.generator.GenerateResponse(schemaRef)
			if err != nil {
				r.logger.Warn("⚠️ Failed to generate response from schema, using fallback",
					"error", err)
				response = r.createFallbackResponse(methodInfo, pathInfo)
			} else {
				response = generatedResp
			}
		} else {
			response = r.createFallbackResponse(methodInfo, pathInfo)
		}
	} else {
		response = r.createFallbackResponse(methodInfo, pathInfo)
	}

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

// createFallbackResponse creates a simple fallback response when schema generation fails
func (r *Router) createFallbackResponse(methodInfo openapi.MethodInfo, pathInfo openapi.PathInfo) map[string]interface{} {
	return map[string]interface{}{
		"message":      "Loki mock response",
		"method":       methodInfo.Method,
		"path":         pathInfo.Path,
		"operation_id": methodInfo.OperationID,
		"timestamp":    "2024-09-07T12:00:00Z",
	}
}

// getOperation retrieves the operation from the spec
func (r *Router) getOperation(path, method string) *openapi3.Operation {
	if r.spec == nil || r.spec.Paths == nil {
		return nil
	}

	pathItem := r.spec.Paths.Find(path)
	if pathItem == nil {
		return nil
	}

	switch method {
	case "GET":
		return pathItem.Get
	case "POST":
		return pathItem.Post
	case "PUT":
		return pathItem.Put
	case "PATCH":
		return pathItem.Patch
	case "DELETE":
		return pathItem.Delete
	case "HEAD":
		return pathItem.Head
	case "OPTIONS":
		return pathItem.Options
	default:
		return nil
	}
}

// getResponseSchema retrieves the response schema for a given status code
func (r *Router) getResponseSchema(operation *openapi3.Operation, statusCode int) *openapi3.Schema {
	if operation == nil || operation.Responses == nil {
		r.logger.Debug("❌ No operation or responses", "has_operation", operation != nil)
		return nil
	}

	// Try the specific status code first
	statusCodeStr := fmt.Sprintf("%d", statusCode)
	responseRef := operation.Responses.Status(statusCode)
	if responseRef == nil {
		r.logger.Debug("⚠️ No response for status code, trying default", "status_code", statusCodeStr)
		// Try default response
		responseRef = operation.Responses.Default()
	}

	if responseRef == nil || responseRef.Value == nil {
		r.logger.Debug("❌ No response ref or value", "has_ref", responseRef != nil)
		return nil
	}

	// Get the response content for application/json
	content := responseRef.Value.Content
	if content == nil {
		r.logger.Debug("❌ No content in response")
		return nil
	}

	mediaType := content.Get("application/json")
	if mediaType == nil {
		r.logger.Debug("❌ No application/json media type")
		return nil
	}

	if mediaType.Schema == nil || mediaType.Schema.Value == nil {
		r.logger.Debug("❌ No schema or schema value", "has_schema", mediaType.Schema != nil)
		return nil
	}

	r.logger.Debug("✅ Using response schema",
		"status_code", statusCodeStr,
		"schema_type", mediaType.Schema.Value.Type)

	return mediaType.Schema.Value
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
