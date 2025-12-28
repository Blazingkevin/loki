package config

import (
	"fmt"
	"regexp"
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate validates the configuration.
func (c *Config) Validate() []ValidationError {
	var errors []ValidationError

	// Validate scenarios
	if len(c.Scenarios) == 0 {
		errors = append(errors, ValidationError{
			Field:   "scenarios",
			Message: "at least one scenario is required",
		})
	}

	// Validate each scenario
	for i, scenario := range c.Scenarios {
		prefix := fmt.Sprintf("scenarios[%d]", i)
		errors = append(errors, validateScenario(scenario, prefix)...)
	}

	// Validate settings
	errors = append(errors, validateSettings(c.Settings)...)

	return errors
}

// validateScenario validates a scenario.
func validateScenario(s Scenario, prefix string) []ValidationError {
	var errors []ValidationError

	// Name is required
	if s.Name == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".name",
			Message: "name is required",
		})
	}

	// At least one trigger is required
	if len(s.Triggers) == 0 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".triggers",
			Message: "at least one trigger is required",
		})
	}

	// Validate triggers
	for i, trigger := range s.Triggers {
		triggerPrefix := fmt.Sprintf("%s.triggers[%d]", prefix, i)
		errors = append(errors, validateTrigger(trigger, triggerPrefix)...)
	}

	// Validate chaos configuration
	errors = append(errors, validateChaos(s.Chaos, prefix+".chaos")...)

	return errors
}

// validateTrigger validates a trigger.
func validateTrigger(t Trigger, prefix string) []ValidationError {
	var errors []ValidationError

	// At least one trigger type must be specified
	hasPath := len(t.Paths) > 0
	hasSchedule := t.Schedule != ""
	hasCondition := t.Condition != ""

	if !hasPath && !hasSchedule && !hasCondition {
		errors = append(errors, ValidationError{
			Field:   prefix,
			Message: "at least one of paths, schedule, or condition must be specified",
		})
	}

	// Validate probability if specified
	if t.Probability < 0 || t.Probability > 1 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".probability",
			Message: "probability must be between 0 and 1",
		})
	}

	// Validate schedule format (basic check)
	if t.Schedule != "" {
		if err := validateCronSchedule(t.Schedule); err != nil {
			errors = append(errors, ValidationError{
				Field:   prefix + ".schedule",
				Message: err.Error(),
			})
		}
	}

	return errors
}

// validateChaos validates chaos configuration.
func validateChaos(c Chaos, prefix string) []ValidationError {
	var errors []ValidationError

	// At least one chaos type must be specified
	hasChaos := c.Latency != nil || c.FailureRate > 0 || len(c.Errors) > 0 || len(c.ResponseCorruption) > 0

	if !hasChaos {
		errors = append(errors, ValidationError{
			Field:   prefix,
			Message: "at least one chaos type must be specified",
		})
	}

	// Validate latency
	if c.Latency != nil {
		errors = append(errors, validateLatency(*c.Latency, prefix+".latency")...)
	}

	// Validate failure rate
	if c.FailureRate < 0 || c.FailureRate > 1 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".failure_rate",
			Message: "failure_rate must be between 0 and 1",
		})
	}

	// Validate errors
	for i, err := range c.Errors {
		errorPrefix := fmt.Sprintf("%s.errors[%d]", prefix, i)
		errors = append(errors, validateError(err, errorPrefix)...)
	}

	// Validate response corruption
	for i, corruption := range c.ResponseCorruption {
		corruptionPrefix := fmt.Sprintf("%s.response_corruption[%d]", prefix, i)
		errors = append(errors, validateCorruption(corruption, corruptionPrefix)...)
	}

	return errors
}

// validateLatency validates latency configuration.
func validateLatency(l LatencyConfig, prefix string) []ValidationError {
	var errors []ValidationError

	// Check if using min/max or base/multiplier pattern
	hasMinMax := l.Min != "" || l.Max != ""
	hasBaseMultiplier := l.Base != "" || l.Multiplier > 0

	if !hasMinMax && !hasBaseMultiplier {
		errors = append(errors, ValidationError{
			Field:   prefix,
			Message: "either min/max or base/multiplier must be specified",
		})
	}

	// Validate that min < max if both specified
	if l.MinDuration > 0 && l.MaxDuration > 0 && l.MinDuration >= l.MaxDuration {
		errors = append(errors, ValidationError{
			Field:   prefix,
			Message: "min duration must be less than max duration",
		})
	}

	// Validate distribution
	validDistributions := map[string]bool{
		"uniform":     true,
		"exponential": true,
		"normal":      true,
	}

	if l.Distribution != "" && !validDistributions[l.Distribution] {
		errors = append(errors, ValidationError{
			Field:   prefix + ".distribution",
			Message: "distribution must be one of: uniform, exponential, normal",
		})
	}

	return errors
}

// validateError validates error configuration.
func validateError(e ErrorConfig, prefix string) []ValidationError {
	var errors []ValidationError

	// HTTP status code must be valid
	if e.Code < 400 || e.Code > 599 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".code",
			Message: "HTTP status code must be between 400 and 599",
		})
	}

	// Probability must be valid
	if e.Probability < 0 || e.Probability > 1 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".probability",
			Message: "probability must be between 0 and 1",
		})
	}

	return errors
}

// validateCorruption validates response corruption configuration.
func validateCorruption(c ResponseCorruptionConfig, prefix string) []ValidationError {
	var errors []ValidationError

	// Type is required
	validTypes := map[string]bool{
		"remove_fields": true,
		"malform_json":  true,
		"truncate":      true,
		"modify_values": true,
	}

	if !validTypes[c.Type] {
		errors = append(errors, ValidationError{
			Field:   prefix + ".type",
			Message: "type must be one of: remove_fields, malform_json, truncate, modify_values",
		})
	}

	// Validate fields for remove_fields type
	if c.Type == "remove_fields" && len(c.Fields) == 0 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".fields",
			Message: "fields are required for remove_fields type",
		})
	}

	// Probability must be valid
	if c.Probability < 0 || c.Probability > 1 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".probability",
			Message: "probability must be between 0 and 1",
		})
	}

	return errors
}

// validateSettings validates global settings.
func validateSettings(s Settings) []ValidationError {
	var errors []ValidationError

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if s.LogLevel != "" && !validLogLevels[s.LogLevel] {
		errors = append(errors, ValidationError{
			Field:   "settings.log_level",
			Message: "log_level must be one of: debug, info, warn, error",
		})
	}

	// Validate dashboard port
	if s.DashboardPort < 0 || s.DashboardPort > 65535 {
		errors = append(errors, ValidationError{
			Field:   "settings.dashboard_port",
			Message: "dashboard_port must be between 0 and 65535",
		})
	}

	return errors
}

// validateCronSchedule validates a cron schedule expression (basic validation).
func validateCronSchedule(schedule string) error {
	if schedule == "" {
		return fmt.Errorf("schedule cannot be empty")
	}

	// Basic validation: check if it has at least 5 fields or is a special format
	// We'll be permissive here and let the actual cron library handle detailed validation
	parts := regexp.MustCompile(`\s+`).Split(schedule, -1)

	// Check for special schedules like @hourly, @daily, etc.
	if len(parts) == 1 && schedule != "" && schedule[0] == '@' {
		return nil
	} // Standard cron has 5-7 fields (minute hour day month weekday [year] [seconds])
	if len(parts) < 5 {
		return fmt.Errorf("invalid cron schedule format: expected at least 5 fields, got %d", len(parts))
	}

	return nil
}
