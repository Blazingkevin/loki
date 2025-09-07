package openapi

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type OpenAPIAnalyzerTestSuite struct {
	suite.Suite
}

func TestOpenAPIAnalyzerTestSuite(t *testing.T) {
	suite.Run(t, new(OpenAPIAnalyzerTestSuite))
}

func (s *OpenAPIAnalyzerTestSuite) TestAnalyzeSpec() {
	parser := NewParser()
	spec, err := parser.LoadSpec("../../test/petstore.yaml")
	assert.NoError(s.T(), err, "Failed to load test spec")

	info := AnalyzeSpec(spec)

	assert.Equal(s.T(), "Sample Pet Store API", info.Title)
	assert.Equal(s.T(), "1.0.0", info.Version)

	assert.NotEmpty(s.T(), info.ServerURLs, "Expected at least one server URL")

	expectedServer := "http://localhost:8080"
	assert.Contains(s.T(), info.ServerURLs, expectedServer)

	assert.Equal(s.T(), 2, info.PathCount)
	assert.Len(s.T(), info.Paths, 2)

	pathMap := make(map[string]PathInfo)
	for _, path := range info.Paths {
		pathMap[path.Path] = path
	}

	petsPath, exists := pathMap["/pets"]
	assert.True(s.T(), exists, "Expected /pets path not found")

	if exists {
		methodMap := make(map[string]MethodInfo)
		for _, method := range petsPath.Methods {
			methodMap[method.Method] = method
		}

		_, hasGet := methodMap["GET"]
		assert.True(s.T(), hasGet, "Expected GET method on /pets")

		_, hasPost := methodMap["POST"]
		assert.True(s.T(), hasPost, "Expected POST method on /pets")

		if getMethod, ok := methodMap["GET"]; ok {
			assert.Equal(s.T(), "listPets", getMethod.OperationID)
		}
	}

	petPath, exists := pathMap["/pets/{petId}"]
	assert.True(s.T(), exists, "Expected /pets/{petId} path not found")

	if exists {
		found := false
		for _, method := range petPath.Methods {
			if method.Method == "GET" && method.OperationID == "getPet" {
				found = true
				break
			}
		}
		assert.True(s.T(), found, "Expected GET method with operationId 'getPet' on /pets/{petId}")
	}

	assert.Equal(s.T(), 3, info.Components.SchemaCount)

	expectedSchemas := []string{"Pet", "NewPet", "Error"}
	for _, expected := range expectedSchemas {
		assert.Contains(s.T(), info.Components.SchemaNames, expected)
	}
}

func (s *OpenAPIAnalyzerTestSuite) TestAnalyzeSpecWithEmptySpec() {
	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Empty API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	info := AnalyzeSpec(spec)

	assert.Equal(s.T(), "Empty API", info.Title)
	assert.Equal(s.T(), 0, info.PathCount)
	assert.Empty(s.T(), info.Paths)
}

func (s *OpenAPIAnalyzerTestSuite) TestAnalyzeSpecWithNilFields() {
	spec := &openapi3.T{
		OpenAPI: "3.0.0",
	}

	info := AnalyzeSpec(spec)

	assert.Empty(s.T(), info.Title)
	assert.Equal(s.T(), 0, info.PathCount)
	assert.Empty(s.T(), info.ServerURLs)
}

func (s *OpenAPIAnalyzerTestSuite) TestExtractPathInfo() {
	paths := openapi3.NewPaths()

	getResponses := openapi3.NewResponses()
	getResponses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: strPtr("Success"),
		},
	})

	postResponses := openapi3.NewResponses()
	postResponses.Set("201", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: strPtr("Created"),
		},
	})

	pathItem := &openapi3.PathItem{
		Summary:     "Test path",
		Description: "A test path for loki unit testing",
		Get: &openapi3.Operation{
			OperationID: "getTest",
			Summary:     "Get test data",
			Tags:        []string{"test"},
			Responses:   getResponses,
		},
		Post: &openapi3.Operation{
			OperationID: "createTest",
			Summary:     "Create test data",
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Description: "Test data",
				},
			},
			Responses: postResponses,
		},
	}

	paths.Set("/test", pathItem)

	pathInfos := extractPathInfo(paths)

	assert.Len(s.T(), pathInfos, 1)

	pathInfo := pathInfos[0]
	assert.Equal(s.T(), "/test", pathInfo.Path)
	assert.Equal(s.T(), "Test path", pathInfo.Summary)
	assert.Len(s.T(), pathInfo.Methods, 2)

	methodMap := make(map[string]MethodInfo)
	for _, method := range pathInfo.Methods {
		methodMap[method.Method] = method
	}

	getMethod, hasGet := methodMap["GET"]
	assert.True(s.T(), hasGet, "Expected GET method")
	if hasGet {
		assert.Equal(s.T(), "getTest", getMethod.OperationID)
		assert.False(s.T(), getMethod.RequestBody, "GET method should not have request body")
		assert.NotEmpty(s.T(), getMethod.Responses, "Expected at least one response code")
		assert.Contains(s.T(), getMethod.Responses, "200")
	}

	postMethod, hasPost := methodMap["POST"]
	assert.True(s.T(), hasPost, "Expected POST method")
	if hasPost {
		assert.Equal(s.T(), "createTest", postMethod.OperationID)
		assert.True(s.T(), postMethod.RequestBody, "POST method should have request body")
	}
}

func (s *OpenAPIAnalyzerTestSuite) TestExtractComponentsInfo() {
	components := &openapi3.Components{
		Schemas: openapi3.Schemas{
			"User":    &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()},
			"Product": &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()},
		},
		SecuritySchemes: openapi3.SecuritySchemes{
			"apiKey": &openapi3.SecuritySchemeRef{
				Value: &openapi3.SecurityScheme{Type: "apiKey"},
			},
		},
		Parameters: openapi3.ParametersMap{
			"limit": &openapi3.ParameterRef{
				Value: &openapi3.Parameter{Name: "limit"},
			},
		},
		Responses: make(map[string]*openapi3.ResponseRef),
	}

	components.Responses["NotFound"] = &openapi3.ResponseRef{
		Value: &openapi3.Response{Description: strPtr("Not Found")},
	}

	info := extractComponentsInfo(components)

	assert.Equal(s.T(), 2, info.SchemaCount)
	assert.Equal(s.T(), 1, info.SecurityCount)
	assert.Equal(s.T(), 1, info.ParameterCount)
	assert.Equal(s.T(), 1, info.ResponseCount)

	expectedSchemas := []string{"User", "Product"}
	for _, expected := range expectedSchemas {
		assert.Contains(s.T(), info.SchemaNames, expected)
	}
}

func strPtr(s string) *string {
	return &s
}
