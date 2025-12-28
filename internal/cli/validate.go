package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Blazingkevin/loki/internal/openapi"
)

var (
	validateOutputFormat string
	validateShowWarnings bool
)

var validateCmd = &cobra.Command{
	Use:   "validate <openapi-spec>",
	Short: "Validate an OpenAPI specification",
	Long: `Validate an OpenAPI specification file for correctness and compatibility.

This command checks:
• Valid YAML syntax
• OpenAPI 3.0+ format compliance  
• Schema definitions completeness
• Endpoint definitions validity
• Reference resolution

Examples:
  # Validate a local spec file
  loki validate petstore.yaml

  # Validate a remote spec
  loki validate https://api.example.com/openapi.yaml
  
  # Output as JSON
  loki validate petstore.yaml --output json
  
  # Hide warnings
  loki validate petstore.yaml --no-warnings`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	specFile := args[0]

	// Create parser and validator
	parser := openapi.NewParser()
	validator := openapi.NewValidator(parser)

	// Perform validation
	result, err := validator.Validate(specFile)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Output based on format
	switch validateOutputFormat {
	case "json":
		return outputJSON(cmd, result)
	default:
		return outputText(result, specFile)
	}
}

func outputJSON(cmd *cobra.Command, result *openapi.ValidationResult) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputText(result *openapi.ValidationResult, specFile string) error {
	fmt.Printf("🔍 Validating OpenAPI specification...\n")
	fmt.Printf("📄 File: %s\n\n", specFile)

	// Display validation status
	if result.Valid {
		fmt.Println("✅ Valid OpenAPI specification")
	} else {
		fmt.Println("❌ Invalid OpenAPI specification")
	}

	// Display stats
	fmt.Println("\n📊 Statistics:")
	fmt.Printf("  Title:              %s\n", result.Stats.Title)
	fmt.Printf("  Version:            %s\n", result.Stats.Version)
	fmt.Printf("  Endpoints:          %d\n", result.Stats.Endpoints)
	fmt.Printf("  Schemas:            %d\n", result.Stats.Schemas)
	fmt.Printf("  Security Schemes:   %d\n", result.Stats.SecuritySchemes)
	fmt.Printf("  Servers:            %d\n", result.Stats.Servers)

	// Display paths
	if len(result.Stats.Paths) > 0 {
		fmt.Println("\n📍 Paths:")
		for path, methods := range result.Stats.Paths {
			fmt.Printf("  %s: %v\n", path, methods)
		}
	}

	// Display errors
	if len(result.Errors) > 0 {
		fmt.Printf("\n❌ Errors (%d):\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  • [%s] %s\n", err.Location, err.Message)
		}
	}

	// Display warnings (if enabled)
	if validateShowWarnings && len(result.Warnings) > 0 {
		fmt.Printf("\n⚠️  Warnings (%d):\n", len(result.Warnings))
		for _, warn := range result.Warnings {
			fmt.Printf("  • [%s] %s\n", warn.Location, warn.Message)
		}
	} else if !validateShowWarnings && len(result.Warnings) > 0 {
		fmt.Printf("\n⚠️  %d warning(s) found (use --warnings to show)\n", len(result.Warnings))
	}

	// Display unused schemas
	if len(result.Stats.UnusedSchemas) > 0 {
		fmt.Printf("\n🗑️  Unused Schemas (%d):\n", len(result.Stats.UnusedSchemas))
		for _, schema := range result.Stats.UnusedSchemas {
			fmt.Printf("  • %s\n", schema)
		}
	}

	// Final status
	fmt.Println()
	if result.Valid {
		fmt.Println("✨ Validation completed successfully!")
		return nil
	}

	return fmt.Errorf("validation failed with %d error(s)", len(result.Errors))
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringVarP(&validateOutputFormat, "output", "o", "text", "Output format (text, json)")
	validateCmd.Flags().BoolVarP(&validateShowWarnings, "warnings", "w", true, "Show validation warnings")
}
