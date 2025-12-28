package openapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator_ValidateValidSpec(t *testing.T) {
	parser := NewParser()
	validator := NewValidator(parser)

	result, err := validator.Validate("../../test/petstore.yaml")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Valid, "Expected spec to be valid")
	assert.Greater(t, result.Stats.Endpoints, 0, "Expected at least one endpoint")
}

func TestValidator_ValidateInvalidFile(t *testing.T) {
	parser := NewParser()
	validator := NewValidator(parser)

	result, err := validator.Validate("nonexistent.yaml")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Valid, "Expected invalid result for nonexistent file")
	assert.NotEmpty(t, result.Errors, "Expected errors for nonexistent file")
}

func TestValidator_CollectStats(t *testing.T) {
	parser := NewParser()
	validator := NewValidator(parser)

	result, err := validator.Validate("../../test/petstore.yaml")
	require.NoError(t, err)
	require.NotNil(t, result)

	stats := result.Stats
	assert.NotEmpty(t, stats.Title, "Expected title to be set")
	assert.NotEmpty(t, stats.Version, "Expected version to be set")
	assert.Greater(t, stats.Endpoints, 0, "Expected at least one endpoint")
	assert.NotEmpty(t, stats.Paths, "Expected at least one path")
}

func TestValidator_CheckUnusedSchemas(t *testing.T) {
	parser := NewParser()
	validator := NewValidator(parser)

	result, err := validator.Validate("../../test/test-validation.yaml")
	require.NoError(t, err)
	assert.NotNil(t, result)

	// This spec has 3 unused schemas
	assert.NotEmpty(t, result.Stats.UnusedSchemas, "Expected unused schemas to be detected")
	assert.GreaterOrEqual(t, len(result.Stats.UnusedSchemas), 3, "Expected at least 3 unused schemas")

	// Check that warnings were generated
	unusedWarningCount := 0
	for _, warning := range result.Warnings {
		if strings.Contains(warning.Message, "never used") {
			unusedWarningCount++
		}
	}
	assert.GreaterOrEqual(t, unusedWarningCount, 3, "Expected warnings for unused schemas")
}

func TestValidator_CheckMissingDescriptions(t *testing.T) {
	parser := NewParser()
	validator := NewValidator(parser)

	result, err := validator.Validate("../../test/test-validation.yaml")
	require.NoError(t, err)
	assert.NotNil(t, result)

	// This spec has endpoints without descriptions
	assert.NotEmpty(t, result.Stats.MissingDescriptions, "Expected missing descriptions to be detected")

	// Check that warnings were generated
	descWarningCount := 0
	for _, warning := range result.Warnings {
		if strings.Contains(warning.Message, "Missing description") {
			descWarningCount++
		}
	}
	assert.Greater(t, descWarningCount, 0, "Expected warnings for missing descriptions")
}

func TestValidator_CheckSecurityDefinitions(t *testing.T) {
	parser := NewParser()
	validator := NewValidator(parser)

	result, err := validator.Validate("../../test/test-invalid-security.yaml")
	require.NoError(t, err)
	assert.NotNil(t, result)

	// This spec references undefined security schemes - should have errors
	assert.False(t, result.Valid, "Expected spec to be invalid due to undefined security schemes")
	assert.NotEmpty(t, result.Errors, "Expected errors for undefined security schemes")

	// Check that at least 2 security errors were found
	securityErrorCount := 0
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "undefined security scheme") {
			securityErrorCount++
		}
	}
	assert.GreaterOrEqual(t, securityErrorCount, 2, "Expected errors for undefined security schemes")
}

func TestValidator_CheckResponseSchemas(t *testing.T) {
	parser := NewParser()
	validator := NewValidator(parser)

	result, err := validator.Validate("../../test/test-validation.yaml")
	require.NoError(t, err)
	assert.NotNil(t, result)

	// This spec has success responses without schemas
	hasResponseWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning.Message, "missing schema") {
			hasResponseWarning = true
			break
		}
	}
	assert.True(t, hasResponseWarning, "Expected warning for missing response schema")
}

func TestExtractSchemaName(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "valid reference",
			ref:      "#/components/schemas/Pet",
			expected: "Pet",
		},
		{
			name:     "valid reference with nested path",
			ref:      "#/components/schemas/User/Address",
			expected: "Address",
		},
		{
			name:     "empty reference",
			ref:      "",
			expected: "",
		},
		{
			name:     "malformed reference",
			ref:      "Pet",
			expected: "Pet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSchemaName(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidationResult_Structure(t *testing.T) {
	result := &ValidationResult{
		Valid: true,
		Errors: []ValidationError{
			{Location: "test", Message: "error"},
		},
		Warnings: []ValidationWarning{
			{Location: "test", Message: "warning"},
		},
		Stats: ValidationStats{
			Version:   "1.0.0",
			Title:     "Test API",
			Endpoints: 5,
			Schemas:   3,
			Paths: map[string][]string{
				"/users": {"GET", "POST"},
			},
		},
	}

	assert.True(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, "1.0.0", result.Stats.Version)
	assert.Equal(t, "Test API", result.Stats.Title)
	assert.Equal(t, 5, result.Stats.Endpoints)
	assert.Equal(t, 3, result.Stats.Schemas)
	assert.Contains(t, result.Stats.Paths, "/users")
}
