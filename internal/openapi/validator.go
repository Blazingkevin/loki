package openapi

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ValidationResult contains the results of OpenAPI spec validation.
type ValidationResult struct {
	Valid    bool
	Errors   []ValidationError
	Warnings []ValidationWarning
	Stats    ValidationStats
}

// ValidationError represents a validation error.
type ValidationError struct {
	Location string
	Message  string
}

// ValidationWarning represents a validation warning.
type ValidationWarning struct {
	Location string
	Message  string
}

// ValidationStats contains statistics about the validated spec.
type ValidationStats struct {
	Version             string
	Title               string
	Endpoints           int
	Schemas             int
	SecuritySchemes     int
	Servers             int
	Paths               map[string][]string // path -> methods
	UnusedSchemas       []string
	MissingDescriptions []string
}

// Validator provides comprehensive OpenAPI spec validation.
type Validator struct {
	parser *Parser
}

// NewValidator creates a new validator.
func NewValidator(parser *Parser) *Validator {
	return &Validator{
		parser: parser,
	}
}

// Validate performs comprehensive validation of an OpenAPI spec.
func (v *Validator) Validate(source string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
		Stats: ValidationStats{
			Paths: make(map[string][]string),
		},
	}

	// Load the spec
	spec, err := v.parser.LoadSpec(source)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Location: "file",
			Message:  fmt.Sprintf("Failed to load spec: %v", err),
		})
		return result, nil
	}

	// Validate the spec using kin-openapi's built-in validation
	if err := v.parser.ValidateSpec(spec); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Location: "spec",
			Message:  fmt.Sprintf("Spec validation failed: %v", err),
		})
	}

	// Collect stats
	v.collectStats(spec, &result.Stats)

	// Perform additional checks
	v.checkUnusedSchemas(spec, result)
	v.checkMissingDescriptions(spec, result)
	v.checkSecurityDefinitions(spec, result)
	v.checkResponseSchemas(spec, result)

	return result, nil
}

// collectStats gathers statistics about the OpenAPI spec.
func (v *Validator) collectStats(spec *openapi3.T, stats *ValidationStats) {
	if spec.Info != nil {
		stats.Version = spec.Info.Version
		stats.Title = spec.Info.Title
	}

	if spec.Components != nil && spec.Components.Schemas != nil {
		stats.Schemas = len(spec.Components.Schemas)
	}

	if spec.Components != nil && spec.Components.SecuritySchemes != nil {
		stats.SecuritySchemes = len(spec.Components.SecuritySchemes)
	}

	if spec.Servers != nil {
		stats.Servers = len(spec.Servers)
	}

	if spec.Paths != nil {
		for path, pathItem := range spec.Paths.Map() {
			methods := []string{}
			if pathItem.Get != nil {
				methods = append(methods, "GET")
			}
			if pathItem.Post != nil {
				methods = append(methods, "POST")
			}
			if pathItem.Put != nil {
				methods = append(methods, "PUT")
			}
			if pathItem.Patch != nil {
				methods = append(methods, "PATCH")
			}
			if pathItem.Delete != nil {
				methods = append(methods, "DELETE")
			}
			if pathItem.Head != nil {
				methods = append(methods, "HEAD")
			}
			if pathItem.Options != nil {
				methods = append(methods, "OPTIONS")
			}
			stats.Paths[path] = methods
			stats.Endpoints += len(methods)
		}
	}
}

// checkUnusedSchemas checks for schemas that are defined but never referenced.
func (v *Validator) checkUnusedSchemas(spec *openapi3.T, result *ValidationResult) {
	if spec.Components == nil || spec.Components.Schemas == nil {
		return
	}

	usedSchemas := make(map[string]bool)

	// Track schemas used in paths
	if spec.Paths != nil {
		for _, pathItem := range spec.Paths.Map() {
			v.trackSchemasInOperation(pathItem.Get, usedSchemas)
			v.trackSchemasInOperation(pathItem.Post, usedSchemas)
			v.trackSchemasInOperation(pathItem.Put, usedSchemas)
			v.trackSchemasInOperation(pathItem.Patch, usedSchemas)
			v.trackSchemasInOperation(pathItem.Delete, usedSchemas)
			v.trackSchemasInOperation(pathItem.Head, usedSchemas)
			v.trackSchemasInOperation(pathItem.Options, usedSchemas)
		}
	}

	// Check for unused schemas
	for schemaName := range spec.Components.Schemas {
		if !usedSchemas[schemaName] {
			result.Stats.UnusedSchemas = append(result.Stats.UnusedSchemas, schemaName)
			result.Warnings = append(result.Warnings, ValidationWarning{
				Location: fmt.Sprintf("components.schemas.%s", schemaName),
				Message:  "Schema defined but never used",
			})
		}
	}
}

// trackSchemasInOperation tracks all schema references in an operation.
func (v *Validator) trackSchemasInOperation(op *openapi3.Operation, usedSchemas map[string]bool) {
	if op == nil {
		return
	}

	// Check request body
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		for _, content := range op.RequestBody.Value.Content {
			if content.Schema != nil {
				v.trackSchemaRef(content.Schema, usedSchemas)
			}
		}
	}

	// Check responses
	if op.Responses != nil {
		for _, resp := range op.Responses.Map() {
			if resp.Value != nil {
				for _, content := range resp.Value.Content {
					if content.Schema != nil {
						v.trackSchemaRef(content.Schema, usedSchemas)
					}
				}
			}
		}
	}
}

// trackSchemaRef recursively tracks schema references including nested ones.
func (v *Validator) trackSchemaRef(schemaRef *openapi3.SchemaRef, usedSchemas map[string]bool) {
	if schemaRef == nil {
		return
	}

	// Track direct reference
	if schemaRef.Ref != "" {
		schemaName := extractSchemaName(schemaRef.Ref)
		if schemaName != "" {
			usedSchemas[schemaName] = true
		}
	}

	// Track nested schemas
	if schemaRef.Value != nil {
		schema := schemaRef.Value

		// Check array items
		if schema.Items != nil {
			v.trackSchemaRef(schema.Items, usedSchemas)
		}

		// Check properties
		for _, propSchema := range schema.Properties {
			v.trackSchemaRef(propSchema, usedSchemas)
		}

		// Check allOf, anyOf, oneOf
		for _, s := range schema.AllOf {
			v.trackSchemaRef(s, usedSchemas)
		}
		for _, s := range schema.AnyOf {
			v.trackSchemaRef(s, usedSchemas)
		}
		for _, s := range schema.OneOf {
			v.trackSchemaRef(s, usedSchemas)
		}

		// Check additionalProperties
		if schema.AdditionalProperties.Schema != nil {
			v.trackSchemaRef(schema.AdditionalProperties.Schema, usedSchemas)
		}
	}
}

// extractSchemaName extracts the schema name from a reference.
func extractSchemaName(ref string) string {
	// Reference format: #/components/schemas/SchemaName
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// checkMissingDescriptions checks for endpoints missing descriptions.
func (v *Validator) checkMissingDescriptions(spec *openapi3.T, result *ValidationResult) {
	if spec.Paths == nil {
		return
	}

	for path, pathItem := range spec.Paths.Map() {
		v.checkOperationDescription(path, "GET", pathItem.Get, result)
		v.checkOperationDescription(path, "POST", pathItem.Post, result)
		v.checkOperationDescription(path, "PUT", pathItem.Put, result)
		v.checkOperationDescription(path, "PATCH", pathItem.Patch, result)
		v.checkOperationDescription(path, "DELETE", pathItem.Delete, result)
		v.checkOperationDescription(path, "HEAD", pathItem.Head, result)
		v.checkOperationDescription(path, "OPTIONS", pathItem.Options, result)
	}
}

// checkOperationDescription checks if an operation has a description.
func (v *Validator) checkOperationDescription(path, method string, op *openapi3.Operation, result *ValidationResult) {
	if op == nil {
		return
	}

	if op.Description == "" && op.Summary == "" {
		location := fmt.Sprintf("%s %s", method, path)
		result.Stats.MissingDescriptions = append(result.Stats.MissingDescriptions, location)
		result.Warnings = append(result.Warnings, ValidationWarning{
			Location: location,
			Message:  "Missing description or summary",
		})
	}
}

// checkSecurityDefinitions checks for security issues.
func (v *Validator) checkSecurityDefinitions(spec *openapi3.T, result *ValidationResult) {
	if spec.Paths == nil {
		return
	}

	// Check if any endpoint uses security that isn't defined
	for path, pathItem := range spec.Paths.Map() {
		v.checkOperationSecurity(path, "GET", pathItem.Get, spec, result)
		v.checkOperationSecurity(path, "POST", pathItem.Post, spec, result)
		v.checkOperationSecurity(path, "PUT", pathItem.Put, spec, result)
		v.checkOperationSecurity(path, "PATCH", pathItem.Patch, spec, result)
		v.checkOperationSecurity(path, "DELETE", pathItem.Delete, spec, result)
		v.checkOperationSecurity(path, "HEAD", pathItem.Head, spec, result)
		v.checkOperationSecurity(path, "OPTIONS", pathItem.Options, spec, result)
	}
}

// checkOperationSecurity validates security requirements for an operation.
func (v *Validator) checkOperationSecurity(path, method string, op *openapi3.Operation, spec *openapi3.T, result *ValidationResult) {
	if op == nil || op.Security == nil {
		return
	}

	definedSchemes := make(map[string]bool)
	if spec.Components != nil && spec.Components.SecuritySchemes != nil {
		for name := range spec.Components.SecuritySchemes {
			definedSchemes[name] = true
		}
	}

	for _, secReq := range *op.Security {
		for schemeName := range secReq {
			if !definedSchemes[schemeName] {
				result.Errors = append(result.Errors, ValidationError{
					Location: fmt.Sprintf("%s %s", method, path),
					Message:  fmt.Sprintf("References undefined security scheme: %s", schemeName),
				})
				result.Valid = false
			}
		}
	}
}

// checkResponseSchemas checks for missing response schemas.
func (v *Validator) checkResponseSchemas(spec *openapi3.T, result *ValidationResult) {
	if spec.Paths == nil {
		return
	}

	for path, pathItem := range spec.Paths.Map() {
		v.checkOperationResponses(path, "POST", pathItem.Post, result)
		v.checkOperationResponses(path, "PUT", pathItem.Put, result)
		v.checkOperationResponses(path, "PATCH", pathItem.Patch, result)
	}
}

// checkOperationResponses checks if success responses have schemas.
func (v *Validator) checkOperationResponses(path, method string, op *openapi3.Operation, result *ValidationResult) {
	if op == nil || op.Responses == nil {
		return
	}

	// Check for 201 (Created) and 200 (OK) responses
	for status, resp := range op.Responses.Map() {
		if (status == "200" || status == "201") && resp.Value != nil {
			hasSchema := false
			for _, content := range resp.Value.Content {
				if content.Schema != nil {
					hasSchema = true
					break
				}
			}

			if !hasSchema {
				result.Warnings = append(result.Warnings, ValidationWarning{
					Location: fmt.Sprintf("%s %s (response %s)", method, path, status),
					Message:  "Success response missing schema definition",
				})
			}
		}
	}
}
