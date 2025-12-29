package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure.
type Config struct {
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description"`
	Scenarios    []Scenario        `yaml:"scenarios"`
	Settings     Settings          `yaml:"settings"`
	FieldMapping map[string]string `yaml:"field_mapping"` // Map field names to faker types
	TypeMapping  map[string]string `yaml:"type_mapping"`  // Map schema types/formats to faker types
}

// Scenario represents a chaos engineering scenario.
type Scenario struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Enabled     bool      `yaml:"enabled"`
	Triggers    []Trigger `yaml:"triggers"`
	Chaos       Chaos     `yaml:"chaos"`
}

// Trigger defines when chaos should be applied.
type Trigger struct {
	Paths       []string `yaml:"paths"`
	Methods     []string `yaml:"methods"`
	Probability float64  `yaml:"probability"`
	Schedule    string   `yaml:"schedule"`
	Duration    string   `yaml:"duration"`
	Condition   string   `yaml:"condition"`
}

// Chaos defines the chaos to apply.
type Chaos struct {
	Latency            *LatencyConfig             `yaml:"latency"`
	FailureRate        float64                    `yaml:"failure_rate"`
	Errors             []ErrorConfig              `yaml:"errors"`
	ResponseCorruption []ResponseCorruptionConfig `yaml:"response_corruption"`
}

// LatencyConfig configures latency injection.
type LatencyConfig struct {
	Min          string  `yaml:"min"`
	Max          string  `yaml:"max"`
	Base         string  `yaml:"base"`
	Multiplier   float64 `yaml:"multiplier"`
	Distribution string  `yaml:"distribution"`

	// Parsed values
	MinDuration  time.Duration `yaml:"-"`
	MaxDuration  time.Duration `yaml:"-"`
	BaseDuration time.Duration `yaml:"-"`
}

// ErrorConfig configures error injection.
type ErrorConfig struct {
	Code        int     `yaml:"code"`
	Message     string  `yaml:"message"`
	Probability float64 `yaml:"probability"`
}

// ResponseCorruptionConfig configures response corruption.
type ResponseCorruptionConfig struct {
	Type        string   `yaml:"type"`
	Fields      []string `yaml:"fields"`
	Probability float64  `yaml:"probability"`
}

// Settings represents global configuration settings.
type Settings struct {
	LogLevel       string `yaml:"log_level"`
	MetricsEnabled bool   `yaml:"metrics_enabled"`
	DashboardPort  int    `yaml:"dashboard_port"`
}

// Load loads configuration from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Parse duration strings
	if err := config.parseDurations(); err != nil {
		return nil, err
	}

	// Set defaults
	config.setDefaults()

	return &config, nil
}

// parseDurations parses duration strings into time.Duration values.
func (c *Config) parseDurations() error {
	for i := range c.Scenarios {
		scenario := &c.Scenarios[i]

		if scenario.Chaos.Latency != nil {
			latency := scenario.Chaos.Latency

			if latency.Min != "" {
				d, err := time.ParseDuration(latency.Min)
				if err != nil {
					return fmt.Errorf("invalid min duration in scenario %s: %w", scenario.Name, err)
				}
				latency.MinDuration = d
			}

			if latency.Max != "" {
				d, err := time.ParseDuration(latency.Max)
				if err != nil {
					return fmt.Errorf("invalid max duration in scenario %s: %w", scenario.Name, err)
				}
				latency.MaxDuration = d
			}

			if latency.Base != "" {
				d, err := time.ParseDuration(latency.Base)
				if err != nil {
					return fmt.Errorf("invalid base duration in scenario %s: %w", scenario.Name, err)
				}
				latency.BaseDuration = d
			}
		}
	}

	return nil
}

// setDefaults sets default values for optional fields.
func (c *Config) setDefaults() {
	// Default settings
	if c.Settings.LogLevel == "" {
		c.Settings.LogLevel = "info"
	}
	if c.Settings.DashboardPort == 0 {
		c.Settings.DashboardPort = 8081
	}

	// Default scenario values
	for i := range c.Scenarios {
		scenario := &c.Scenarios[i]

		// Enable by default if not specified
		if !scenario.Enabled {
			scenario.Enabled = true
		}

		// Default error probabilities to 1.0 if not specified
		for j := range scenario.Chaos.Errors {
			if scenario.Chaos.Errors[j].Probability == 0 {
				scenario.Chaos.Errors[j].Probability = 1.0
			}
		}

		// Default corruption probabilities to 1.0 if not specified
		for j := range scenario.Chaos.ResponseCorruption {
			if scenario.Chaos.ResponseCorruption[j].Probability == 0 {
				scenario.Chaos.ResponseCorruption[j].Probability = 1.0
			}
		}

		// Default latency distribution
		if scenario.Chaos.Latency != nil && scenario.Chaos.Latency.Distribution == "" {
			scenario.Chaos.Latency.Distribution = "uniform"
		}
	}
}

// GetScenario returns a scenario by name.
func (c *Config) GetScenario(name string) *Scenario {
	for i := range c.Scenarios {
		if c.Scenarios[i].Name == name {
			return &c.Scenarios[i]
		}
	}
	return nil
}

// GetEnabledScenarios returns all enabled scenarios.
func (c *Config) GetEnabledScenarios() []Scenario {
	enabled := []Scenario{}
	for _, scenario := range c.Scenarios {
		if scenario.Enabled {
			enabled = append(enabled, scenario)
		}
	}
	return enabled
}
