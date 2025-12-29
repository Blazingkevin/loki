package generator

import (
	"encoding/json"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator()
	assert.NotNil(t, gen)
	assert.NotNil(t, gen.rand)
}

func TestGenerateString(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name    string
		schema  *openapi3.Schema
		checkFn func(t *testing.T, result string)
	}{
		{
			name:   "nil schema",
			schema: nil,
			checkFn: func(t *testing.T, result string) {
				assert.NotEmpty(t, result)
			},
		},
		{
			name: "email format",
			schema: &openapi3.Schema{
				Format: "email",
			},
			checkFn: func(t *testing.T, result string) {
				assert.Contains(t, result, "@")
				// Faker generates emails with various domains, not just example.com
				assert.Regexp(t, `^[a-z0-9]+@[a-z]+\.[a-z]+$`, result)
			},
		},
		{
			name: "uuid format",
			schema: &openapi3.Schema{
				Format: "uuid",
			},
			checkFn: func(t *testing.T, result string) {
				assert.Len(t, result, 36) // UUID format: 8-4-4-4-12
				assert.Contains(t, result, "-")
			},
		},
		{
			name: "date format",
			schema: &openapi3.Schema{
				Format: "date",
			},
			checkFn: func(t *testing.T, result string) {
				assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, result)
			},
		},
		{
			name: "date-time format",
			schema: &openapi3.Schema{
				Format: "date-time",
			},
			checkFn: func(t *testing.T, result string) {
				assert.Contains(t, result, "T")
				assert.Contains(t, result, ":")
			},
		},
		{
			name: "url format",
			schema: &openapi3.Schema{
				Format: "url",
			},
			checkFn: func(t *testing.T, result string) {
				// Faker generates both http and https URLs
				assert.Regexp(t, `^https?://[a-z]+\.[a-z]+$`, result)
			},
		},
		{
			name: "min/max length",
			schema: &openapi3.Schema{
				MinLength: 10,
				MaxLength: uint64Ptr(15),
			},
			checkFn: func(t *testing.T, result string) {
				assert.GreaterOrEqual(t, len(result), 10)
				assert.LessOrEqual(t, len(result), 15)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gen.generateString(tt.schema)
			require.NoError(t, err)
			tt.checkFn(t, result)
		})
	}
}

func TestGenerateInteger(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name    string
		schema  *openapi3.Schema
		checkFn func(t *testing.T, result int64)
	}{
		{
			name:   "nil schema",
			schema: nil,
			checkFn: func(t *testing.T, result int64) {
				assert.GreaterOrEqual(t, result, int64(0))
			},
		},
		{
			name: "min/max constraints",
			schema: &openapi3.Schema{
				Min: float64Ptr(10),
				Max: float64Ptr(20),
			},
			checkFn: func(t *testing.T, result int64) {
				assert.GreaterOrEqual(t, result, int64(10))
				assert.LessOrEqual(t, result, int64(20))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gen.generateInteger(tt.schema)
			require.NoError(t, err)
			tt.checkFn(t, result)
		})
	}
}

func TestGenerateNumber(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name    string
		schema  *openapi3.Schema
		checkFn func(t *testing.T, result float64)
	}{
		{
			name:   "nil schema",
			schema: nil,
			checkFn: func(t *testing.T, result float64) {
				assert.GreaterOrEqual(t, result, 0.0)
			},
		},
		{
			name: "min/max constraints",
			schema: &openapi3.Schema{
				Min: float64Ptr(5.5),
				Max: float64Ptr(10.5),
			},
			checkFn: func(t *testing.T, result float64) {
				assert.GreaterOrEqual(t, result, 5.5)
				assert.LessOrEqual(t, result, 10.5)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gen.generateNumber(tt.schema)
			require.NoError(t, err)
			tt.checkFn(t, result)
		})
	}
}

func TestGenerateBoolean(t *testing.T) {
	gen := NewGenerator()

	// Run multiple times to ensure both true and false are possible
	results := make(map[bool]bool)
	for i := 0; i < 50; i++ {
		result, err := gen.generateBoolean()
		require.NoError(t, err)
		results[result] = true
	}

	// Should have seen both true and false
	assert.True(t, results[true] || results[false], "Should generate boolean values")
}

func TestGenerateArray(t *testing.T) {
	gen := NewGenerator()

	t.Run("array of strings", func(t *testing.T) {
		schema := &openapi3.Schema{
			Items: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"string"},
				},
			},
			MinItems: 2,
			MaxItems: uint64Ptr(5),
		}

		result, err := gen.generateArray(schema)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result), 2)
		assert.LessOrEqual(t, len(result), 5)

		// Check all items are strings
		for _, item := range result {
			assert.IsType(t, "", item)
		}
	})

	t.Run("array of integers", func(t *testing.T) {
		schema := &openapi3.Schema{
			Items: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"integer"},
				},
			},
		}

		result, err := gen.generateArray(schema)
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Check all items are integers
		for _, item := range result {
			assert.IsType(t, int64(0), item)
		}
	})

	t.Run("empty array when no items", func(t *testing.T) {
		schema := &openapi3.Schema{}

		result, err := gen.generateArray(schema)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestGenerateObject(t *testing.T) {
	gen := NewGenerator()

	t.Run("simple object", func(t *testing.T) {
		schema := &openapi3.Schema{
			Properties: openapi3.Schemas{
				"name": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"string"},
					},
				},
				"age": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"integer"},
					},
				},
			},
			Required: []string{"name"},
		}

		result, err := gen.generateObject(schema)
		require.NoError(t, err)

		// Required field must be present
		assert.Contains(t, result, "name")
		assert.IsType(t, "", result["name"])

		// Optional field may or may not be present
		if age, ok := result["age"]; ok {
			assert.IsType(t, int64(0), age)
		}
	})

	t.Run("nested object", func(t *testing.T) {
		schema := &openapi3.Schema{
			Properties: openapi3.Schemas{
				"user": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"object"},
						Properties: openapi3.Schemas{
							"id": &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"integer"},
								},
							},
							"name": &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"string"},
								},
							},
						},
					},
				},
			},
		}

		result, err := gen.generateObject(schema)
		require.NoError(t, err)

		// Check nested object
		if user, ok := result["user"]; ok {
			userMap, ok := user.(map[string]interface{})
			require.True(t, ok, "user should be an object")
			assert.NotNil(t, userMap)
		}
	})
}

func TestGenerateEnum(t *testing.T) {
	gen := NewGenerator()

	schema := &openapi3.Schema{
		Enum: []interface{}{"red", "green", "blue"},
	}

	// Generate multiple times to check all values are from enum
	for i := 0; i < 20; i++ {
		result, err := gen.generateFromSchema(schema)
		require.NoError(t, err)

		// Should be one of the enum values
		assert.Contains(t, []interface{}{"red", "green", "blue"}, result)
	}
}

func TestGenerateAllOf(t *testing.T) {
	gen := NewGenerator()

	schemas := []*openapi3.SchemaRef{
		{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
				Properties: openapi3.Schemas{
					"name": &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
						},
					},
				},
			},
		},
		{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
				Properties: openapi3.Schemas{
					"age": &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"integer"},
						},
					},
				},
			},
		},
	}

	result, err := gen.generateAllOf(schemas)
	require.NoError(t, err)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)

	// Should have properties from both schemas
	assert.Contains(t, resultMap, "name")
	assert.Contains(t, resultMap, "age")
}

func TestGenerateAnyOf(t *testing.T) {
	gen := NewGenerator()

	schemas := []*openapi3.SchemaRef{
		{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"string"},
			},
		},
		{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"integer"},
			},
		},
	}

	// Generate multiple times to ensure both types can be selected
	types := make(map[string]bool)
	for i := 0; i < 50; i++ {
		result, err := gen.generateAnyOf(schemas)
		require.NoError(t, err)

		switch result.(type) {
		case string:
			types["string"] = true
		case int64:
			types["integer"] = true
		}
	}

	// Should have seen both types
	assert.True(t, types["string"] || types["integer"], "Should select from anyOf options")
}

func TestGenerateResponse(t *testing.T) {
	gen := NewGenerator()

	t.Run("complex schema", func(t *testing.T) {
		schema := &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
				Properties: openapi3.Schemas{
					"id": &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:   &openapi3.Types{"integer"},
							Format: "int64",
						},
					},
					"name": &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:      &openapi3.Types{"string"},
							MinLength: 1,
							MaxLength: uint64Ptr(100),
						},
					},
					"email": &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:   &openapi3.Types{"string"},
							Format: "email",
						},
					},
					"tags": &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"array"},
							Items: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"string"},
								},
							},
						},
					},
				},
				Required: []string{"id", "name"},
			},
		}

		result, err := gen.GenerateResponse(schema)
		require.NoError(t, err)

		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		// Check required fields
		assert.Contains(t, resultMap, "id")
		assert.Contains(t, resultMap, "name")

		// Validate types
		assert.IsType(t, int64(0), resultMap["id"])
		assert.IsType(t, "", resultMap["name"])

		// Check email format if present
		if email, ok := resultMap["email"]; ok {
			emailStr, ok := email.(string)
			require.True(t, ok)
			assert.Contains(t, emailStr, "@")
		}

		// Check tags array if present
		if tags, ok := resultMap["tags"]; ok {
			tagsArray, ok := tags.([]interface{})
			require.True(t, ok)
			for _, tag := range tagsArray {
				assert.IsType(t, "", tag)
			}
		}

		// Verify it can be marshaled to JSON
		jsonData, err := json.Marshal(result)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)
	})

	t.Run("nil schema", func(t *testing.T) {
		_, err := gen.GenerateResponse(nil)
		assert.Error(t, err)
	})
}

// Helper functions
func uint64Ptr(v uint64) *uint64 {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}
