package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ValidConfig(t *testing.T) {
	config, err := Load("../../configs/example-chaos.yaml")
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "Example Chaos Configuration", config.Name)
	assert.NotEmpty(t, config.Description)
	assert.Greater(t, len(config.Scenarios), 0)
}

func TestLoad_NonexistentFile(t *testing.T) {
	config, err := Load("nonexistent.yaml")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create temporary invalid YAML file
	tmpFile, err := os.CreateTemp("", "invalid-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("invalid: yaml: content: [")
	require.NoError(t, err)
	tmpFile.Close()

	config, err := Load(tmpFile.Name())
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse config")
}

func TestParseDurations(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{
			{
				Name: "test",
				Chaos: Chaos{
					Latency: &LatencyConfig{
						Min:  "100ms",
						Max:  "2s",
						Base: "50ms",
					},
				},
			},
		},
	}

	err := config.parseDurations()
	require.NoError(t, err)

	latency := config.Scenarios[0].Chaos.Latency
	assert.Equal(t, 100*time.Millisecond, latency.MinDuration)
	assert.Equal(t, 2*time.Second, latency.MaxDuration)
	assert.Equal(t, 50*time.Millisecond, latency.BaseDuration)
}

func TestParseDurations_InvalidFormat(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{
			{
				Name: "test",
				Chaos: Chaos{
					Latency: &LatencyConfig{
						Min: "invalid",
					},
				},
			},
		},
	}

	err := config.parseDurations()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid min duration")
}

func TestSetDefaults(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{
			{
				Name: "test",
				Chaos: Chaos{
					Latency: &LatencyConfig{},
					Errors: []ErrorConfig{
						{Code: 500},
					},
					ResponseCorruption: []ResponseCorruptionConfig{
						{Type: "remove_fields"},
					},
				},
			},
		},
	}

	config.setDefaults()

	// Check default settings
	assert.Equal(t, "info", config.Settings.LogLevel)
	assert.Equal(t, 8081, config.Settings.DashboardPort)

	// Check scenario defaults
	assert.True(t, config.Scenarios[0].Enabled)
	assert.Equal(t, "uniform", config.Scenarios[0].Chaos.Latency.Distribution)
	assert.Equal(t, 1.0, config.Scenarios[0].Chaos.Errors[0].Probability)
	assert.Equal(t, 1.0, config.Scenarios[0].Chaos.ResponseCorruption[0].Probability)
}

func TestGetScenario(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{
			{Name: "scenario1"},
			{Name: "scenario2"},
		},
	}

	scenario := config.GetScenario("scenario2")
	require.NotNil(t, scenario)
	assert.Equal(t, "scenario2", scenario.Name)

	scenario = config.GetScenario("nonexistent")
	assert.Nil(t, scenario)
}

func TestGetEnabledScenarios(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{
			{Name: "enabled1", Enabled: true},
			{Name: "disabled", Enabled: false},
			{Name: "enabled2", Enabled: true},
		},
	}

	enabled := config.GetEnabledScenarios()
	assert.Len(t, enabled, 2)
	assert.Equal(t, "enabled1", enabled[0].Name)
	assert.Equal(t, "enabled2", enabled[1].Name)
}

func TestValidate_ValidConfig(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{
			{
				Name: "test",
				Triggers: []Trigger{
					{
						Paths:       []string{"/api/*"},
						Probability: 0.5,
					},
				},
				Chaos: Chaos{
					Latency: &LatencyConfig{
						Min:          "100ms",
						Max:          "500ms",
						MinDuration:  100 * time.Millisecond,
						MaxDuration:  500 * time.Millisecond,
						Distribution: "uniform",
					},
				},
			},
		},
		Settings: Settings{
			LogLevel:      "info",
			DashboardPort: 8081,
		},
	}

	errors := config.Validate()
	assert.Empty(t, errors, "Expected no validation errors")
}

func TestValidate_NoScenarios(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{},
	}

	errors := config.Validate()
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0].Error(), "at least one scenario is required")
}

func TestValidate_InvalidProbability(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{
			{
				Name: "test",
				Triggers: []Trigger{
					{
						Paths:       []string{"/api/*"},
						Probability: 1.5, // Invalid: > 1
					},
				},
				Chaos: Chaos{
					Latency: &LatencyConfig{},
				},
			},
		},
	}

	errors := config.Validate()
	assert.NotEmpty(t, errors)
	found := false
	for _, err := range errors {
		if contains(err.Error(), "probability must be between 0 and 1") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected probability validation error")
}

func TestValidate_InvalidHTTPCode(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{
			{
				Name: "test",
				Triggers: []Trigger{
					{Paths: []string{"/api/*"}},
				},
				Chaos: Chaos{
					Errors: []ErrorConfig{
						{Code: 200}, // Invalid: not an error code
					},
				},
			},
		},
	}

	errors := config.Validate()
	assert.NotEmpty(t, errors)
	found := false
	for _, err := range errors {
		if contains(err.Error(), "HTTP status code must be between 400 and 599") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected HTTP code validation error")
}

func TestValidate_InvalidDistribution(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{
			{
				Name: "test",
				Triggers: []Trigger{
					{Paths: []string{"/api/*"}},
				},
				Chaos: Chaos{
					Latency: &LatencyConfig{
						Min:          "100ms",
						Max:          "500ms",
						MinDuration:  100 * time.Millisecond,
						MaxDuration:  500 * time.Millisecond,
						Distribution: "invalid",
					},
				},
			},
		},
	}

	errors := config.Validate()
	assert.NotEmpty(t, errors)
	found := false
	for _, err := range errors {
		if contains(err.Error(), "distribution must be one of") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected distribution validation error")
}

func TestValidate_MinGreaterThanMax(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{
			{
				Name: "test",
				Triggers: []Trigger{
					{Paths: []string{"/api/*"}},
				},
				Chaos: Chaos{
					Latency: &LatencyConfig{
						Min:         "2s",
						Max:         "1s",
						MinDuration: 2 * time.Second,
						MaxDuration: 1 * time.Second,
					},
				},
			},
		},
	}

	errors := config.Validate()
	assert.NotEmpty(t, errors)
	found := false
	for _, err := range errors {
		if contains(err.Error(), "min duration must be less than max duration") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected min/max validation error")
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	config := &Config{
		Scenarios: []Scenario{
			{
				Name:     "test",
				Triggers: []Trigger{{Paths: []string{"/"}}},
				Chaos:    Chaos{FailureRate: 0.5},
			},
		},
		Settings: Settings{
			LogLevel: "invalid",
		},
	}

	errors := config.Validate()
	assert.NotEmpty(t, errors)
	found := false
	for _, err := range errors {
		if contains(err.Error(), "log_level must be one of") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected log level validation error")
}

func TestValidateCronSchedule(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		valid    bool
	}{
		{"valid standard cron", "0 */2 * * *", true},
		{"valid with days", "* * * * SAT,SUN", true},
		{"valid simple", "* * * * *", true},
		{"valid every N", "*/15 * * * *", true},
		{"invalid empty", "", false},
		{"invalid format", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCronSchedule(tt.schedule)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// Helper function.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
