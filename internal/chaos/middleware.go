package chaos

import (
	"encoding/json"
	"net/http"

	"github.com/Blazingkevin/loki/internal/server"
)

// Middleware integrates chaos engineering into the HTTP request flow
func Middleware(engine *Engine, logger *server.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Evaluate if chaos should be applied
			decision := engine.Evaluate(r)

			if !decision.ShouldApply {
				// No chaos - proceed normally
				next.ServeHTTP(w, r)
				return
			}

			// Get request ID for logging
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = w.Header().Get("X-Request-ID")
			}

			// Log chaos application
			logger.LogChaosApplied(
				requestID,
				decision.Scenario.Name,
				decision.Action.Type,
				decision.Action.Details,
			)

			// Apply the appropriate chaos action
			switch decision.Action.Type {
			case "latency":
				// Apply latency before processing the request
				if err := engine.ApplyLatency(&decision.Action); err != nil {
					logger.LogError(err, requestID, "applying latency chaos")
				}
				next.ServeHTTP(w, r)

			case "error":
				// Return error response instead of processing request
				code, _ := decision.Action.Details["code"].(int)
				message, _ := decision.Action.Details["message"].(string)

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Chaos-Applied", "true")
				w.Header().Set("X-Chaos-Scenario", decision.Scenario.Name)
				w.WriteHeader(code)

				response := map[string]interface{}{
					"error":   message,
					"code":    code,
					"__chaos": "error_injection",
				}
				json.NewEncoder(w).Encode(response)

			case "corruption":
				// For corruption, we'd need to intercept the response
				// For now, we'll add a header indicating corruption should be applied
				// The actual corruption will be handled by response interceptor
				w.Header().Set("X-Chaos-Corruption", "pending")
				w.Header().Set("X-Chaos-Scenario", decision.Scenario.Name)

				// Store corruption details in context for response processing
				// TODO: Implement response body interception and corruption
				next.ServeHTTP(w, r)

			default:
				// Unknown chaos type - log and proceed
				logger.Warn("unknown chaos action type",
					"type", decision.Action.Type,
					"request_id", requestID,
				)
				next.ServeHTTP(w, r)
			}
		})
	}
}
