package chaos

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blazingkevin/loki/internal/config"
)

func TestNewEngine(t *testing.T) {
	cfg := &config.Config{
		Name: "test",
	}

	engine := NewEngine(cfg)
	assert.NotNil(t, engine)
	assert.Equal(t, cfg, engine.config)
	assert.NotNil(t, engine.rand)
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{
			name:     "exact match",
			path:     "/api/users",
			pattern:  "/api/users",
			expected: true,
		},
		{
			name:     "wildcard suffix match",
			path:     "/api/users",
			pattern:  "/api/*",
			expected: true,
		},
		{
			name:     "wildcard suffix no match",
			path:     "/other/users",
			pattern:  "/api/*",
			expected: false,
		},
		{
			name:     "wildcard middle match",
			path:     "/api/123/items",
			pattern:  "/api/*/items",
			expected: true,
		},
		{
			name:     "wildcard middle no match",
			path:     "/api/123/other",
			pattern:  "/api/*/items",
			expected: false,
		},
		{
			name:     "no match",
			path:     "/api/users",
			pattern:  "/api/products",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPath(tt.path, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{
			name:     "single wildcard match",
			path:     "/api/123/items",
			pattern:  "/api/*/items",
			expected: true,
		},
		{
			name:     "multiple wildcards match",
			path:     "/api/123/items/456",
			pattern:  "/api/*/items/*",
			expected: true,
		},
		{
			name:     "length mismatch",
			path:     "/api/123",
			pattern:  "/api/*/items",
			expected: false,
		},
		{
			name:     "segment mismatch",
			path:     "/api/123/products",
			pattern:  "/api/*/items",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchWildcard(tt.path, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchesTrigger(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name     string
		request  *http.Request
		trigger  *config.Trigger
		expected bool
	}{
		{
			name:    "path match",
			request: httptest.NewRequest("GET", "/api/users", nil),
			trigger: &config.Trigger{
				Paths: []string{"/api/*"},
			},
			expected: true,
		},
		{
			name:    "path no match",
			request: httptest.NewRequest("GET", "/other/users", nil),
			trigger: &config.Trigger{
				Paths: []string{"/api/*"},
			},
			expected: false,
		},
		{
			name:    "method match",
			request: httptest.NewRequest("POST", "/api/users", nil),
			trigger: &config.Trigger{
				Methods: []string{"POST", "PUT"},
			},
			expected: true,
		},
		{
			name:    "method no match",
			request: httptest.NewRequest("GET", "/api/users", nil),
			trigger: &config.Trigger{
				Methods: []string{"POST", "PUT"},
			},
			expected: false,
		},
		{
			name:    "path and method match",
			request: httptest.NewRequest("POST", "/api/users", nil),
			trigger: &config.Trigger{
				Paths:   []string{"/api/*"},
				Methods: []string{"POST"},
			},
			expected: true,
		},
		{
			name:    "path match but method no match",
			request: httptest.NewRequest("GET", "/api/users", nil),
			trigger: &config.Trigger{
				Paths:   []string{"/api/*"},
				Methods: []string{"POST"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.matchesTrigger(tt.request, tt.trigger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluate(t *testing.T) {
	t.Run("nil config returns no decision", func(t *testing.T) {
		engine := NewEngine(nil)
		req := httptest.NewRequest("GET", "/api/users", nil)

		decision := engine.Evaluate(req)
		assert.False(t, decision.ShouldApply)
	})

	t.Run("no enabled scenarios returns no decision", func(t *testing.T) {
		cfg := &config.Config{
			Scenarios: []config.Scenario{
				{
					Name:    "test",
					Enabled: false,
				},
			},
		}
		engine := NewEngine(cfg)
		req := httptest.NewRequest("GET", "/api/users", nil)

		decision := engine.Evaluate(req)
		assert.False(t, decision.ShouldApply)
	})

	t.Run("matching scenario with high probability", func(t *testing.T) {
		cfg := &config.Config{
			Scenarios: []config.Scenario{
				{
					Name:    "latency_test",
					Enabled: true,
					Triggers: []config.Trigger{
						{
							Paths:       []string{"/api/*"},
							Probability: 1.0, // Always apply
						},
					},
					Chaos: config.Chaos{
						Latency: &config.LatencyConfig{
							MinDuration:  50 * time.Millisecond,
							MaxDuration:  100 * time.Millisecond,
							Distribution: "uniform",
						},
					},
				},
			},
		}
		engine := NewEngine(cfg)
		req := httptest.NewRequest("GET", "/api/users", nil)

		decision := engine.Evaluate(req)
		assert.True(t, decision.ShouldApply)
		assert.Equal(t, "latency_test", decision.Scenario.Name)
		assert.Equal(t, "latency", decision.Action.Type)
	})

	t.Run("non-matching path returns no decision", func(t *testing.T) {
		cfg := &config.Config{
			Scenarios: []config.Scenario{
				{
					Name:    "test",
					Enabled: true,
					Triggers: []config.Trigger{
						{
							Paths:       []string{"/api/*"},
							Probability: 1.0,
						},
					},
					Chaos: config.Chaos{
						Latency: &config.LatencyConfig{
							MinDuration:  50 * time.Millisecond,
							MaxDuration:  100 * time.Millisecond,
							Distribution: "uniform",
						},
					},
				},
			},
		}
		engine := NewEngine(cfg)
		req := httptest.NewRequest("GET", "/other/path", nil)

		decision := engine.Evaluate(req)
		assert.False(t, decision.ShouldApply)
	})
}

func TestCreateLatencyAction(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name         string
		config       *config.LatencyConfig
		expectNil    bool
		checkDelay   bool
		minDelay     time.Duration
		maxDelay     time.Duration
		distribution string
	}{
		{
			name:      "nil config",
			config:    nil,
			expectNil: true,
		},
		{
			name: "uniform distribution",
			config: &config.LatencyConfig{
				MinDuration:  50 * time.Millisecond,
				MaxDuration:  100 * time.Millisecond,
				Distribution: "uniform",
			},
			checkDelay:   true,
			minDelay:     50 * time.Millisecond,
			maxDelay:     100 * time.Millisecond,
			distribution: "uniform",
		},
		{
			name: "normal distribution",
			config: &config.LatencyConfig{
				MinDuration:  50 * time.Millisecond,
				MaxDuration:  100 * time.Millisecond,
				Distribution: "normal",
			},
			checkDelay:   true,
			minDelay:     50 * time.Millisecond,
			maxDelay:     100 * time.Millisecond,
			distribution: "normal",
		},
		{
			name: "exponential distribution",
			config: &config.LatencyConfig{
				MinDuration:  50 * time.Millisecond,
				MaxDuration:  100 * time.Millisecond,
				Distribution: "exponential",
			},
			checkDelay:   true,
			minDelay:     50 * time.Millisecond,
			maxDelay:     100 * time.Millisecond,
			distribution: "exponential",
		},
		{
			name: "default distribution",
			config: &config.LatencyConfig{
				MinDuration:  50 * time.Millisecond,
				MaxDuration:  100 * time.Millisecond,
				Distribution: "unknown",
			},
			checkDelay:   true,
			minDelay:     50 * time.Millisecond,
			maxDelay:     100 * time.Millisecond,
			distribution: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := engine.createLatencyAction(tt.config)

			if tt.expectNil {
				assert.Nil(t, action)
				return
			}

			require.NotNil(t, action)
			assert.Equal(t, "latency", action.Type)
			assert.Equal(t, tt.distribution, action.Details["distribution"])

			if tt.checkDelay {
				delayMs, ok := action.Details["delay_ms"].(int64)
				require.True(t, ok)
				assert.GreaterOrEqual(t, delayMs, tt.minDelay.Milliseconds())
				assert.LessOrEqual(t, delayMs, tt.maxDelay.Milliseconds())
			}
		})
	}
}

func TestCreateErrorAction(t *testing.T) {
	engine := NewEngine(nil)

	t.Run("empty errors returns nil", func(t *testing.T) {
		action := engine.createErrorAction([]config.ErrorConfig{})
		assert.Nil(t, action)
	})

	t.Run("single error", func(t *testing.T) {
		errors := []config.ErrorConfig{
			{
				Code:        500,
				Message:     "Internal Server Error",
				Probability: 1.0,
			},
		}

		action := engine.createErrorAction(errors)
		require.NotNil(t, action)
		assert.Equal(t, "error", action.Type)
		assert.Equal(t, 500, action.Details["code"])
		assert.Equal(t, "Internal Server Error", action.Details["message"])
	})

	t.Run("multiple errors with probabilities", func(t *testing.T) {
		errors := []config.ErrorConfig{
			{
				Code:        500,
				Message:     "Internal Server Error",
				Probability: 0.5,
			},
			{
				Code:        503,
				Message:     "Service Unavailable",
				Probability: 0.5,
			},
		}

		// Run multiple times to ensure both can be selected
		codes := make(map[int]bool)
		for i := 0; i < 100; i++ {
			action := engine.createErrorAction(errors)
			require.NotNil(t, action)
			assert.Equal(t, "error", action.Type)

			code, ok := action.Details["code"].(int)
			require.True(t, ok)
			codes[code] = true
		}

		// Should have seen both error codes
		assert.True(t, codes[500] || codes[503], "Should select at least one error code")
	})
}

func TestCreateCorruptionAction(t *testing.T) {
	engine := NewEngine(nil)

	t.Run("empty corruptions returns nil", func(t *testing.T) {
		action := engine.createCorruptionAction([]config.ResponseCorruptionConfig{})
		assert.Nil(t, action)
	})

	t.Run("single corruption", func(t *testing.T) {
		corruptions := []config.ResponseCorruptionConfig{
			{
				Type:        "remove_fields",
				Fields:      []string{"id", "name"},
				Probability: 1.0,
			},
		}

		action := engine.createCorruptionAction(corruptions)
		require.NotNil(t, action)
		assert.Equal(t, "corruption", action.Type)
		assert.Equal(t, "remove_fields", action.Details["corruption_type"])
		assert.Equal(t, []string{"id", "name"}, action.Details["fields"])
	})

	t.Run("multiple corruptions with probabilities", func(t *testing.T) {
		corruptions := []config.ResponseCorruptionConfig{
			{
				Type:        "remove_fields",
				Fields:      []string{"id"},
				Probability: 0.5,
			},
			{
				Type:        "modify_values",
				Fields:      []string{"name"},
				Probability: 0.5,
			},
		}

		// Run multiple times to ensure both can be selected
		types := make(map[string]bool)
		for i := 0; i < 100; i++ {
			action := engine.createCorruptionAction(corruptions)
			require.NotNil(t, action)
			assert.Equal(t, "corruption", action.Type)

			cType, ok := action.Details["corruption_type"].(string)
			require.True(t, ok)
			types[cType] = true
		}

		// Should have seen both corruption types
		assert.True(t, types["remove_fields"] || types["modify_values"], "Should select at least one corruption type")
	})
}

func TestApplyLatency(t *testing.T) {
	engine := NewEngine(nil)

	t.Run("invalid action type", func(t *testing.T) {
		action := &Action{
			Type: "error",
		}

		err := engine.ApplyLatency(action)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid action type")
	})

	t.Run("missing delay_ms", func(t *testing.T) {
		action := &Action{
			Type:    "latency",
			Details: map[string]interface{}{},
		}

		err := engine.ApplyLatency(action)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing or invalid delay_ms")
	})

	t.Run("applies latency", func(t *testing.T) {
		action := &Action{
			Type: "latency",
			Details: map[string]interface{}{
				"delay_ms": int64(50),
			},
		}

		start := time.Now()
		err := engine.ApplyLatency(action)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, duration, 50*time.Millisecond)
		assert.LessOrEqual(t, duration, 100*time.Millisecond) // Allow some overhead
	})
}

func TestSelectAction(t *testing.T) {
	engine := NewEngine(nil)

	t.Run("no chaos configured returns nil", func(t *testing.T) {
		scenario := &config.Scenario{
			Chaos: config.Chaos{},
		}

		action := engine.selectAction(scenario)
		assert.Nil(t, action)
	})

	t.Run("selects latency action", func(t *testing.T) {
		scenario := &config.Scenario{
			Chaos: config.Chaos{
				Latency: &config.LatencyConfig{
					MinDuration:  50 * time.Millisecond,
					MaxDuration:  100 * time.Millisecond,
					Distribution: "uniform",
				},
			},
		}

		action := engine.selectAction(scenario)
		require.NotNil(t, action)
		assert.Equal(t, "latency", action.Type)
	})

	t.Run("selects error action", func(t *testing.T) {
		scenario := &config.Scenario{
			Chaos: config.Chaos{
				Errors: []config.ErrorConfig{
					{Code: 500, Message: "Error", Probability: 1.0},
				},
			},
		}

		action := engine.selectAction(scenario)
		require.NotNil(t, action)
		assert.Equal(t, "error", action.Type)
	})

	t.Run("selects corruption action", func(t *testing.T) {
		scenario := &config.Scenario{
			Chaos: config.Chaos{
				ResponseCorruption: []config.ResponseCorruptionConfig{
					{Type: "remove_fields", Fields: []string{"id"}, Probability: 1.0},
				},
			},
		}

		action := engine.selectAction(scenario)
		require.NotNil(t, action)
		assert.Equal(t, "corruption", action.Type)
	})

	t.Run("randomly selects from multiple options", func(t *testing.T) {
		scenario := &config.Scenario{
			Chaos: config.Chaos{
				Latency: &config.LatencyConfig{
					MinDuration:  50 * time.Millisecond,
					MaxDuration:  100 * time.Millisecond,
					Distribution: "uniform",
				},
				Errors: []config.ErrorConfig{
					{Code: 500, Message: "Error", Probability: 1.0},
				},
				ResponseCorruption: []config.ResponseCorruptionConfig{
					{Type: "remove_fields", Fields: []string{"id"}, Probability: 1.0},
				},
			},
		}

		// Run multiple times to ensure randomness
		types := make(map[string]bool)
		for i := 0; i < 50; i++ {
			action := engine.selectAction(scenario)
			require.NotNil(t, action)
			types[action.Type] = true
		}

		// Should have selected at least 2 different types
		assert.GreaterOrEqual(t, len(types), 2, "Should select multiple chaos types randomly")
	})
}
