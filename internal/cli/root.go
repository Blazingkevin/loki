package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	// current version of loki
	Version = "v1.0.0-dev"

	AppName = "loki"

	AppDescription = "API mocking with chaos engineering - The trickster for your APIs"
)

var rootCmd = &cobra.Command{
	Use:   AppName,
	Short: "Loki: The trickster for your APIs",
	Long: fmt.Sprintf(`%s

Loki is an open-source tool that generates realistic mock APIs from OpenAPI 
specifications while introducing configurable chaos engineering patterns.

Named after the Norse trickster god, Loki helps developers test application 
resilience against real-world API failures and network conditions.

🔥 Key Features:
• Generate mock APIs from OpenAPI specs
• Inject chaos patterns (latency, failures, corruption)
• Record and replay real API traffic
• Configure scenarios with YAML
• Docker-ready deployment

For more information, visit: https://github.com/Blazingkevin/loki`, AppDescription),
	Version: Version,
	Example: `  # Start a basic mock server
  loki serve api-spec.yaml

  # Add chaos engineering
  loki serve api-spec.yaml --chaos chaos-config.yaml --port 8080

  # Validate OpenAPI specification
  loki validate api-spec.yaml

  # Show version information
  loki version`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("%s version %s\n", AppName, Version))
}
