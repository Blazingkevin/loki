package openapi

import (
	"maps"

	"github.com/getkin/kin-openapi/openapi3"
)

type SpecInfo struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	ServerURLs  []string       `json:"server_urls"`
	PathCount   int            `json:"path_count"`
	Paths       []PathInfo     `json:"paths"`
	Components  ComponentsInfo `json:"components"`
	Extensions  map[string]any `json:"extenstions,omitempty"`
}

type ComponentsInfo struct {
	SchemaCount    int      `json:"schema_count"`
	SchemaNames    []string `json:"schema_names,omitempty"`
	SecurityCount  int      `json:"security_count"`
	SecurityNames  []string `json:"security_names,omitempty"`
	ParameterCount int      `json:"parameter_count"`
	ParameterNames []string `json:"parameter_names,omitempty"`
	ResponseCount  int      `json:"response_count"`
	ResponseNames  []string `json:"response_names,omitempty"`
}
type PathInfo struct {
	Path        string       `json:"path"`
	Methods     []MethodInfo `json:"methods"`
	Parameters  []string     `json:"parameters,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	Description string       `json:"description,omitempty"`
}

type MethodInfo struct {
	Method      string   `json:"method"`
	OperationID string   `json:"operation_id,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Parameters  []string `json:"parameters,omitempty"`
	RequestBody bool     `json:"has_request_body"`
	Responses   []string `json:"response_codes"`
}

func AnalyzeSpec(spec *openapi3.T) *SpecInfo {
	info := &SpecInfo{
		Extensions: make(map[string]any),
	}

	if spec.Info != nil {
		info.Title = spec.Info.Title
		info.Description = spec.Info.Description
		info.Version = spec.Info.Version

		maps.Copy(info.Extensions, spec.Info.Extensions)
	}

	info.ServerURLs = make([]string, 0, len(spec.Servers))
	for _, server := range spec.Servers {
		if server.URL != "" {
			info.ServerURLs = append(info.ServerURLs, server.URL)
		}
	}

	if spec.Paths != nil {
		info.PathCount = spec.Paths.Len()
		info.Paths = extractPathInfo(spec.Paths)
	}

	if spec.Components != nil {
		info.Components = extractComponentsInfo(spec.Components)
	}

	return info
}

func extractPathInfo(paths *openapi3.Paths) []PathInfo {
	pathInfos := make([]PathInfo, 0, paths.Len())

	pathMap := paths.Map()
	for path, pathItem := range pathMap {
		pathInfo := PathInfo{
			Path:        path,
			Methods:     make([]MethodInfo, 0, 8),
			Summary:     pathItem.Summary,
			Description: pathItem.Description,
		}

		if len(pathItem.Parameters) > 0 {
			pathInfo.Parameters = make([]string, 0, len(pathItem.Parameters))
			for _, param := range pathItem.Parameters {
				if param.Value != nil && param.Value.Name != "" {
					pathInfo.Parameters = append(pathInfo.Parameters, param.Value.Name)
				}
			}
		}

		operations := map[string]*openapi3.Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"DELETE":  pathItem.Delete,
			"PATCH":   pathItem.Patch,
			"HEAD":    pathItem.Head,
			"OPTIONS": pathItem.Options,
			"TRACE":   pathItem.Trace,
		}

		for method, operation := range operations {
			if operation != nil {
				methodInfo := MethodInfo{
					Method:      method,
					OperationID: operation.OperationID,
					Summary:     operation.Summary,
					Description: operation.Description,
					Tags:        operation.Tags,
					RequestBody: operation.RequestBody != nil,
				}

				if len(operation.Parameters) > 0 {
					methodInfo.Parameters = make([]string, 0, len(operation.Parameters))
					for _, param := range operation.Parameters {
						if param.Value != nil && param.Value.Name != "" {
							methodInfo.Parameters = append(methodInfo.Parameters, param.Value.Name)
						}
					}
				}

				if operation.Responses != nil {
					responseMap := operation.Responses.Map()
					methodInfo.Responses = make([]string, 0, len(responseMap))
					for code := range responseMap {
						methodInfo.Responses = append(methodInfo.Responses, code)
					}
				}

				pathInfo.Methods = append(pathInfo.Methods, methodInfo)
			}
		}

		pathInfos = append(pathInfos, pathInfo)
	}

	return pathInfos
}

func extractComponentsInfo(components *openapi3.Components) ComponentsInfo {
	info := ComponentsInfo{}

	if components.Schemas != nil {
		info.SchemaCount = len(components.Schemas)
		info.SchemaNames = make([]string, 0, len(components.Schemas))
		for name := range components.Schemas {
			info.SchemaNames = append(info.SchemaNames, name)
		}
	}

	if components.SecuritySchemes != nil {
		info.SecurityCount = len(components.SecuritySchemes)
		info.SecurityNames = make([]string, 0, len(components.SecuritySchemes))
		for name := range components.SecuritySchemes {
			info.SecurityNames = append(info.SecurityNames, name)
		}
	}

	if components.Parameters != nil {
		info.ParameterCount = len(components.Parameters)
		info.ParameterNames = make([]string, 0, len(components.Parameters))
		for name := range components.Parameters {
			info.ParameterNames = append(info.ParameterNames, name)
		}
	}

	if components.Responses != nil {
		info.ResponseCount = len(components.Responses)
		info.ResponseNames = make([]string, 0, len(components.Responses))
		for name := range components.Responses {
			info.ResponseNames = append(info.ResponseNames, name)
		}
	}

	return info
}
