package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Blazingkevin/loki/internal/chaos"
	"github.com/Blazingkevin/loki/internal/config"
	"github.com/Blazingkevin/loki/internal/openapi"
	"github.com/Blazingkevin/loki/internal/server"
)

var (
	servePort   int
	serveHost   string
	chaosConfig string
	logLevel    string
	profile     string
)

var serveCommand = &cobra.Command{
	Use:   "serve <openapi-spec>",
	Short: "Start a mock API server from OpenAPI specification",
	Long: `Start a mock API server that serves endpoints defined in the openAPI specification.

The server will generate realistic responses based on schema definitions
and can optionally inject chaos patterns to simulate real-world failures.

Examples:
	#Basic mock server
	loki serve petstore.yaml

	#With custom port and host
	loki serve petstore.yaml --port 3000 --host 0.0.0.0

	#With chaos engineering
	loki serve petstore.yaml --chaos chaos-config.yaml 

	#With specific profile
	loki serve petstore.yaml --chaos chaos-config.yaml --profile production,

`,
	Args: cobra.ExactArgs(1),
	RunE: runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	specFile := args[0]

	// Setup logger with configured level
	logger := server.NewLogger(logLevel, os.Stdout)

	fmt.Printf("🔥 Starting Loki mock server...\n")
	fmt.Printf("📋 OpenAPI spec: %s\n", specFile)
	if logLevel != "" {
		fmt.Printf("📊 Log level: %s\n", logLevel)
	}

	parser := openapi.NewParser()
	spec, err := parser.LoadAndValidate(specFile)
	if err != nil {
		fmt.Printf("Failed to load OpenAPI spec: %v\n", err)
		return err
	}

	info := openapi.AnalyzeSpec(spec)

	fmt.Printf("Loaded %s v%s (%d endpoints)\n", info.Title, info.Version, info.PathCount)

	// Load chaos configuration if provided
	var chaosConf *config.Config
	if chaosConfig != "" {
		fmt.Printf("🎭 Chaos config: %s\n", chaosConfig)
		if profile != "" {
			fmt.Printf("📊 Chaos profile: %s\n", profile)
		}

		chaosConf, err = config.Load(chaosConfig)
		if err != nil {
			fmt.Printf("Failed to load chaos config: %v\n", err)
			return err
		}

		// Validate chaos configuration
		validationErrors := chaosConf.Validate()
		if len(validationErrors) > 0 {
			fmt.Printf("Invalid chaos configuration:\n")
			for _, ve := range validationErrors {
				fmt.Printf("  • %s: %s\n", ve.Field, ve.Message)
			}
			return fmt.Errorf("chaos configuration validation failed")
		}

		enabledScenarios := chaosConf.GetEnabledScenarios()
		fmt.Printf("✅ Loaded %d enabled chaos scenarios\n", len(enabledScenarios))
		for _, scenario := range enabledScenarios {
			fmt.Printf("  • %s: %s\n", scenario.Name, scenario.Description)
		}
	}

	fmt.Printf("\nAvailable endpoints:\n")
	for _, path := range info.Paths {
		for _, method := range path.Methods {
			fmt.Printf("  %-6s %s", method.Method, path.Path)
			if method.Summary != "" {
				fmt.Printf(" - %s", method.Summary)
			}
			fmt.Printf("\n")
		}
	}

	serverConfig := &server.Config{
		Host:        serveHost,
		Port:        servePort,
		Spec:        spec,
		SpecInfo:    info,
		Logger:      logger,
		LogLevel:    logLevel,
		ChaosConfig: chaosConf,
	}

	srv, err := server.New(serverConfig)
	if err != nil {
		fmt.Printf("Failed to create server: %v\n", err)
		return err
	}

	// Apply chaos middleware if chaos config is loaded
	if chaosConf != nil {
		chaosEngine := chaos.NewEngine(chaosConf)
		srv.WrapHandler(chaos.Middleware(chaosEngine, logger))
		fmt.Printf("🎭 Chaos engineering activated!\n")
	}

	fmt.Printf("🌐 Server: http://%s:%d\n", serveHost, servePort)
	fmt.Printf("🎯 Health check: http://%s:%d/_loki/health\n", serveHost, servePort)
	fmt.Printf("📋 Spec info: http://%s:%d/_loki/spec\n", serveHost, servePort)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return srv.Start(ctx)
}

func init() {
	rootCmd.AddCommand(serveCommand)

	// Server configuration
	serveCommand.Flags().IntVarP(&servePort, "port", "p", 8080, "Server port")
	serveCommand.Flags().StringVar(&serveHost, "host", "localhost", "Server host")

	// Chaos configuration flags
	serveCommand.Flags().StringVarP(&chaosConfig, "chaos", "c", "", "Chaos configuration file")
	serveCommand.Flags().StringVar(&profile, "profile", "", "Chaos profile")

	// Logging configuration
	serveCommand.Flags().StringVar(&logLevel, "log-level", "info", "Logging level (debug, warn, info, error)")
}
