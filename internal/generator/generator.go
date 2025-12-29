package generator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// Generator creates realistic responses from OpenAPI schemas
type Generator struct {
	rand         *rand.Rand
	faker        *Faker
	fieldMapping map[string]string // Map field names to faker types
	typeMapping  map[string]string // Map schema types/formats to faker types
}

// NewGenerator creates a new response generator
func NewGenerator() *Generator {
	return &Generator{
		rand:         rand.New(rand.NewSource(time.Now().UnixNano())),
		faker:        NewFaker(),
		fieldMapping: make(map[string]string),
		typeMapping:  make(map[string]string),
	}
}

// SetFieldMapping sets custom field name to faker type mapping
func (g *Generator) SetFieldMapping(mapping map[string]string) {
	g.fieldMapping = mapping
}

// SetTypeMapping sets custom schema type/format to faker type mapping
func (g *Generator) SetTypeMapping(mapping map[string]string) {
	g.typeMapping = mapping
}

// GenerateResponse generates a response body from an OpenAPI schema
func (g *Generator) GenerateResponse(schema *openapi3.SchemaRef) (interface{}, error) {
	if schema == nil || schema.Value == nil {
		return nil, fmt.Errorf("schema is nil")
	}

	return g.generateFromSchema(schema.Value)
}

// generateFromSchema generates data matching a schema
func (g *Generator) generateFromSchema(schema *openapi3.Schema) (interface{}, error) {
	// Handle references (should be resolved by loader)
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}

	// Handle allOf, anyOf, oneOf
	if len(schema.AllOf) > 0 {
		return g.generateAllOf(schema.AllOf)
	}
	if len(schema.AnyOf) > 0 {
		return g.generateAnyOf(schema.AnyOf)
	}
	if len(schema.OneOf) > 0 {
		return g.generateOneOf(schema.OneOf)
	}

	// Handle enum
	if len(schema.Enum) > 0 {
		return g.selectEnum(schema.Enum), nil
	}

	// Generate by type
	schemaType := "object" // default
	if schema.Type != nil && len(*schema.Type) > 0 {
		schemaType = (*schema.Type)[0]
	}

	switch schemaType {
	case "object":
		return g.generateObject(schema)
	case "array":
		return g.generateArray(schema)
	case "string":
		return g.generateString(schema)
	case "integer":
		return g.generateInteger(schema)
	case "number":
		return g.generateNumber(schema)
	case "boolean":
		return g.generateBoolean()
	default:
		return nil, fmt.Errorf("unsupported type: %s", schemaType)
	}
}

// generateObject generates an object from schema properties
func (g *Generator) generateObject(schema *openapi3.Schema) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Generate required properties
	requiredMap := make(map[string]bool)
	for _, req := range schema.Required {
		requiredMap[req] = true
	}

	// Generate all properties (or at least required ones)
	for propName, propSchema := range schema.Properties {
		// Always generate required properties
		// Generate optional properties with 70% probability
		if requiredMap[propName] || g.rand.Float64() < 0.7 {
			// Check if there's a field mapping for this property
			if fakerType, ok := g.fieldMapping[propName]; ok && propSchema.Value != nil {
				// Use faker for this field if it's a string type
				schemaType := "object"
				if propSchema.Value.Type != nil && len(*propSchema.Value.Type) > 0 {
					schemaType = (*propSchema.Value.Type)[0]
				}
				if schemaType == "string" {
					result[propName] = g.faker.Generate(fakerType)
					continue
				}
			}

			value, err := g.generateFromSchema(propSchema.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to generate property %s: %w", propName, err)
			}
			result[propName] = value
		}
	}

	// If no properties defined but additionalProperties is allowed, generate some
	if len(schema.Properties) == 0 && schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
		// Generate 1-3 additional properties
		numProps := 1 + g.rand.Intn(3)
		for i := 0; i < numProps; i++ {
			key := fmt.Sprintf("field_%d", i+1)
			if schema.AdditionalProperties.Schema != nil {
				value, err := g.generateFromSchema(schema.AdditionalProperties.Schema.Value)
				if err != nil {
					continue
				}
				result[key] = value
			} else {
				value, _ := g.generateString(nil)
				result[key] = value
			}
		}
	}

	return result, nil
}

// generateArray generates an array from schema items
func (g *Generator) generateArray(schema *openapi3.Schema) ([]interface{}, error) {
	if schema.Items == nil {
		return []interface{}{}, nil
	}

	// Determine array length
	minItems := 1
	maxItems := 5

	if schema.MinItems > 0 {
		minItems = int(schema.MinItems)
	}
	if schema.MaxItems != nil && *schema.MaxItems > 0 {
		maxItems = int(*schema.MaxItems)
	}

	// Ensure maxItems >= minItems
	if maxItems < minItems {
		maxItems = minItems
	}

	// Random length between min and max
	length := minItems
	if maxItems > minItems {
		length = minItems + g.rand.Intn(maxItems-minItems+1)
	}

	result := make([]interface{}, length)
	for i := 0; i < length; i++ {
		item, err := g.generateFromSchema(schema.Items.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to generate array item: %w", err)
		}
		result[i] = item
	}

	return result, nil
}

// generateString generates a string value
func (g *Generator) generateString(schema *openapi3.Schema) (string, error) {
	if schema == nil {
		return "sample string", nil
	}

	// Check for x-faker extension
	if fakerType, ok := schema.Extensions["x-faker"].(string); ok {
		return g.faker.Generate(fakerType), nil
	}

	// Check type mapping for format
	if schema.Format != "" {
		if fakerType, ok := g.typeMapping[schema.Format]; ok {
			return g.faker.Generate(fakerType), nil
		}
	}

	// Check format for specific string types
	switch schema.Format {
	case "date":
		return time.Now().Format("2006-01-02"), nil
	case "date-time":
		return time.Now().Format(time.RFC3339), nil
	case "email":
		return g.faker.Email(), nil
	case "uuid":
		return g.faker.UUID(), nil
	case "uri", "url":
		return g.faker.URL(), nil
	case "hostname":
		return g.faker.DomainName(), nil
	case "ipv4":
		return g.faker.IPv4(), nil
	case "ipv6":
		return g.faker.IPv6(), nil
	}

	// Check pattern constraint
	if schema.Pattern != "" {
		// For now, return a sample that might match common patterns
		// Full regex-based generation would require additional library
		return g.generatePatternString(schema.Pattern), nil
	}

	// Check length constraints
	minLength := 5
	maxLength := 20

	if schema.MinLength > 0 {
		minLength = int(schema.MinLength)
	}
	if schema.MaxLength != nil && *schema.MaxLength > 0 {
		maxLength = int(*schema.MaxLength)
	}

	if maxLength < minLength {
		maxLength = minLength
	}

	// Generate random string
	length := minLength
	if maxLength > minLength {
		length = minLength + g.rand.Intn(maxLength-minLength+1)
	}

	return g.randomString(length), nil
}

// generateInteger generates an integer value
func (g *Generator) generateInteger(schema *openapi3.Schema) (int64, error) {
	if schema == nil {
		return int64(g.rand.Intn(100)), nil
	}

	min := int64(0)
	max := int64(1000)

	if schema.Min != nil {
		min = int64(*schema.Min)
	}
	if schema.Max != nil {
		max = int64(*schema.Max)
	}

	if max < min {
		max = min + 100
	}

	// Generate random integer in range
	return min + g.rand.Int63n(max-min+1), nil
}

// generateNumber generates a floating point number
func (g *Generator) generateNumber(schema *openapi3.Schema) (float64, error) {
	if schema == nil {
		return g.rand.Float64() * 100, nil
	}

	min := 0.0
	max := 1000.0

	if schema.Min != nil {
		min = *schema.Min
	}
	if schema.Max != nil {
		max = *schema.Max
	}

	if max < min {
		max = min + 100
	}

	// Generate random number in range
	return min + g.rand.Float64()*(max-min), nil
}

// generateBoolean generates a boolean value
func (g *Generator) generateBoolean() (bool, error) {
	return g.rand.Float64() < 0.5, nil
}

// generateAllOf merges all schemas in allOf
func (g *Generator) generateAllOf(schemas []*openapi3.SchemaRef) (interface{}, error) {
	// Merge all properties from all schemas
	merged := make(map[string]interface{})

	for _, schemaRef := range schemas {
		if schemaRef.Value == nil {
			continue
		}

		schema := schemaRef.Value

		// Directly generate properties from this schema
		// rather than calling generateFromSchema which might recurse
		for propName, propSchema := range schema.Properties {
			value, err := g.generateFromSchema(propSchema.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to generate property %s: %w", propName, err)
			}
			merged[propName] = value
		}

		// Also handle if this schema has nested allOf
		if len(schema.AllOf) > 0 {
			nestedObj, err := g.generateAllOf(schema.AllOf)
			if err != nil {
				return nil, err
			}
			if objMap, ok := nestedObj.(map[string]interface{}); ok {
				for k, v := range objMap {
					merged[k] = v
				}
			}
		}
	}

	return merged, nil
}

// generateAnyOf picks one schema from anyOf
func (g *Generator) generateAnyOf(schemas []*openapi3.SchemaRef) (interface{}, error) {
	if len(schemas) == 0 {
		return nil, fmt.Errorf("anyOf is empty")
	}

	// Pick random schema
	selected := schemas[g.rand.Intn(len(schemas))]
	return g.generateFromSchema(selected.Value)
}

// generateOneOf picks one schema from oneOf
func (g *Generator) generateOneOf(schemas []*openapi3.SchemaRef) (interface{}, error) {
	if len(schemas) == 0 {
		return nil, fmt.Errorf("oneOf is empty")
	}

	// Pick random schema
	selected := schemas[g.rand.Intn(len(schemas))]
	return g.generateFromSchema(selected.Value)
}

// selectEnum picks a random value from enum
func (g *Generator) selectEnum(enum []interface{}) interface{} {
	if len(enum) == 0 {
		return nil
	}
	return enum[g.rand.Intn(len(enum))]
}

// randomString generates a random string of given length
func (g *Generator) randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[g.rand.Intn(len(charset))]
	}
	return string(b)
}

// generatePatternString generates a string that might match a pattern
func (g *Generator) generatePatternString(pattern string) string {
	// Simple pattern matching for common cases
	// Full implementation would use a regex-to-string library
	switch pattern {
	case "^[0-9]+$":
		return fmt.Sprintf("%d", 1000+g.rand.Intn(9000))
	case "^[a-z]+$":
		return g.randomString(10)
	case "^[A-Z]+$":
		return "SAMPLE"
	default:
		return "sample_" + g.randomString(5)
	}
}
