package chaos

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/Blazingkevin/loki/internal/config"
)

// Engine manages chaos injection for requests
type Engine struct {
	config *config.Config
	rand   *rand.Rand
}

// NewEngine creates a new chaos engine
func NewEngine(cfg *config.Config) *Engine {
	return &Engine{
		config: cfg,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Decision represents the chaos action to take
type Decision struct {
	ShouldApply bool
	Scenario    *config.Scenario
	Action      Action
}

// Action represents a specific chaos action
type Action struct {
	Type    string                 // "latency", "error", "corruption"
	Details map[string]interface{} // Action-specific details
}

// Evaluate determines if chaos should be applied to a request
func (e *Engine) Evaluate(r *http.Request) *Decision {
	if e.config == nil {
		return &Decision{ShouldApply: false}
	}

	// Get enabled scenarios
	scenarios := e.config.GetEnabledScenarios()
	if len(scenarios) == 0 {
		return &Decision{ShouldApply: false}
	}

	// Check each scenario
	for _, scenario := range scenarios {
		if e.shouldApplyScenario(r, &scenario) {
			action := e.selectAction(&scenario)
			if action != nil {
				return &Decision{
					ShouldApply: true,
					Scenario:    &scenario,
					Action:      *action,
				}
			}
		}
	}

	return &Decision{ShouldApply: false}
}

// shouldApplyScenario checks if a scenario should be applied to a request
func (e *Engine) shouldApplyScenario(r *http.Request, scenario *config.Scenario) bool {
	// Check all triggers
	for _, trigger := range scenario.Triggers {
		if e.matchesTrigger(r, &trigger) {
			// Roll dice for probability
			if e.rand.Float64() < trigger.Probability {
				return true
			}
		}
	}
	return false
}

// matchesTrigger checks if a request matches a trigger's conditions
func (e *Engine) matchesTrigger(r *http.Request, trigger *config.Trigger) bool {
	// Check path patterns
	if len(trigger.Paths) > 0 {
		pathMatches := false
		for _, pattern := range trigger.Paths {
			if matchPath(r.URL.Path, pattern) {
				pathMatches = true
				break
			}
		}
		if !pathMatches {
			return false
		}
	}

	// Check methods
	if len(trigger.Methods) > 0 {
		methodMatches := false
		for _, method := range trigger.Methods {
			if strings.EqualFold(r.Method, method) {
				methodMatches = true
				break
			}
		}
		if !methodMatches {
			return false
		}
	}

	return true
}

// matchPath checks if a path matches a pattern (supports wildcards)
func matchPath(path, pattern string) bool {
	// Exact match
	if path == pattern {
		return true
	}

	// Wildcard match: /api/* matches /api/users, /api/products, etc.
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix)
	}

	// Wildcard match: /api/*/items matches /api/123/items, /api/abc/items, etc.
	if strings.Contains(pattern, "*") {
		return matchWildcard(path, pattern)
	}

	return false
}

// matchWildcard performs wildcard pattern matching
func matchWildcard(path, pattern string) bool {
	pathParts := strings.Split(path, "/")
	patternParts := strings.Split(pattern, "/")

	if len(pathParts) != len(patternParts) {
		return false
	}

	for i := range patternParts {
		if patternParts[i] == "*" {
			continue
		}
		if pathParts[i] != patternParts[i] {
			return false
		}
	}

	return true
}

// selectAction chooses which chaos action to apply from a scenario
func (e *Engine) selectAction(scenario *config.Scenario) *Action {
	chaos := scenario.Chaos

	// Count available chaos types
	var options []string
	if chaos.Latency != nil {
		options = append(options, "latency")
	}
	if len(chaos.Errors) > 0 {
		options = append(options, "error")
	}
	if len(chaos.ResponseCorruption) > 0 {
		options = append(options, "corruption")
	}

	if len(options) == 0 {
		return nil
	}

	// Randomly select a chaos type
	chaosType := options[e.rand.Intn(len(options))]

	switch chaosType {
	case "latency":
		return e.createLatencyAction(chaos.Latency)
	case "error":
		return e.createErrorAction(chaos.Errors)
	case "corruption":
		return e.createCorruptionAction(chaos.ResponseCorruption)
	}

	return nil
}

// createLatencyAction creates a latency chaos action
func (e *Engine) createLatencyAction(cfg *config.LatencyConfig) *Action {
	if cfg == nil {
		return nil
	}

	var delay time.Duration

	switch cfg.Distribution {
	case "uniform":
		// Random value between min and max
		diff := cfg.MaxDuration - cfg.MinDuration
		delay = cfg.MinDuration + time.Duration(e.rand.Int63n(int64(diff)))

	case "normal":
		// Normal distribution around mean with standard deviation
		mean := float64(cfg.MinDuration+cfg.MaxDuration) / 2
		stddev := float64(cfg.MaxDuration-cfg.MinDuration) / 6 // 99.7% within range
		delay = time.Duration(e.rand.NormFloat64()*stddev + mean)

		// Clamp to range
		if delay < cfg.MinDuration {
			delay = cfg.MinDuration
		}
		if delay > cfg.MaxDuration {
			delay = cfg.MaxDuration
		}

	case "exponential":
		// Exponential distribution starting from min
		lambda := 1.0 / float64(cfg.MaxDuration-cfg.MinDuration)
		delay = cfg.MinDuration + time.Duration(e.rand.ExpFloat64()/lambda)

		// Clamp to max
		if delay > cfg.MaxDuration {
			delay = cfg.MaxDuration
		}

	default:
		delay = cfg.MinDuration
	}

	return &Action{
		Type: "latency",
		Details: map[string]interface{}{
			"delay_ms":     delay.Milliseconds(),
			"distribution": cfg.Distribution,
		},
	}
}

// createErrorAction creates an error chaos action
func (e *Engine) createErrorAction(errors []config.ErrorConfig) *Action {
	if len(errors) == 0 {
		return nil
	}

	// Calculate total probability
	totalProb := 0.0
	for _, err := range errors {
		totalProb += err.Probability
	}

	// Normalize if needed
	if totalProb == 0 {
		totalProb = 1.0
	}

	// Select error based on probability
	roll := e.rand.Float64() * totalProb
	cumProb := 0.0

	for _, err := range errors {
		cumProb += err.Probability
		if roll <= cumProb {
			return &Action{
				Type: "error",
				Details: map[string]interface{}{
					"code":    err.Code,
					"message": err.Message,
				},
			}
		}
	}

	// Fallback to first error
	return &Action{
		Type: "error",
		Details: map[string]interface{}{
			"code":    errors[0].Code,
			"message": errors[0].Message,
		},
	}
}

// createCorruptionAction creates a response corruption chaos action
func (e *Engine) createCorruptionAction(corruptions []config.ResponseCorruptionConfig) *Action {
	if len(corruptions) == 0 {
		return nil
	}

	// Calculate total probability
	totalProb := 0.0
	for _, c := range corruptions {
		totalProb += c.Probability
	}

	// Normalize if needed
	if totalProb == 0 {
		totalProb = 1.0
	}

	// Select corruption based on probability
	roll := e.rand.Float64() * totalProb
	cumProb := 0.0

	for _, corruption := range corruptions {
		cumProb += corruption.Probability
		if roll <= cumProb {
			return &Action{
				Type: "corruption",
				Details: map[string]interface{}{
					"corruption_type": corruption.Type,
					"fields":          corruption.Fields,
					"probability":     corruption.Probability,
				},
			}
		}
	}

	// Fallback to first corruption
	return &Action{
		Type: "corruption",
		Details: map[string]interface{}{
			"corruption_type": corruptions[0].Type,
			"fields":          corruptions[0].Fields,
			"probability":     corruptions[0].Probability,
		},
	}
}

// ApplyLatency applies latency chaos to a request
func (e *Engine) ApplyLatency(action *Action) error {
	if action.Type != "latency" {
		return fmt.Errorf("invalid action type for latency: %s", action.Type)
	}

	delayMs, ok := action.Details["delay_ms"].(int64)
	if !ok {
		return fmt.Errorf("missing or invalid delay_ms in action")
	}

	time.Sleep(time.Duration(delayMs) * time.Millisecond)
	return nil
}
