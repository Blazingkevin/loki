package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	if rootCmd.Use != AppName {
		t.Errorf("Expected use to be %q, got %q", AppName, rootCmd.Use)
	}

	if rootCmd.Short != "Loki: The Trickster for Your APIs" {
		t.Errorf("Expected Short to be 'Loki: The Trickster for Your APIs', got %q", rootCmd.Short)
	}

	if rootCmd.Version != Version {
		t.Errorf("Expected Version to be %q, got %q", Version, rootCmd.Version)
	}

	expectedCommands := []string{"serve", "validate", "version"}
	commands := rootCmd.Commands()
	commandNames := make([]string, len(commands))

	for i, cmd := range commands {
		commandNames[i] = cmd.Name()
	}

	for _, expected := range expectedCommands {
		found := false

		for _, name := range commandNames {
			if name == expected {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected command %q to be registered", expected)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	var versionCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "version" {
			versionCmd = cmd
			break
		}
	}

	if versionCmd == nil {
		t.Fatal("Version command not found")
	}

	if versionCmd.Short != "Show version and build information" {
		t.Errorf("Expected Short description for version command")
	}

	if versionCmd.Use != "version" {
		t.Errorf("Expected Use to be 'version', got %q", versionCmd.Use)
	}
}

func TestServeCommandValidation(t *testing.T) {
	var serveCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "serve" {
			serveCmd = cmd
			break
		}
	}

	if serveCmd == nil {
		t.Fatal("Serve command not found")
	}

	if serveCmd.Use != "serve <openapi-spec>" {
		t.Errorf("Expected Use to be 'serve <openapi-spec>', got %q", serveCmd.Use)
	}

	if serveCmd.Short != "Start a mock API server from OpenAPI specification" {
		t.Errorf("Expected correct Short description for serve command")
	}

	expectedFlags := []string{"port", "host", "chaos", "profile", "log-level"}
	for _, flagName := range expectedFlags {
		flag := serveCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q to be defined", flagName)
		}
	}
}

func TestValidateCommandValidation(t *testing.T) {
	var validateCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "validate" {
			validateCmd = cmd
			break
		}
	}

	if validateCmd == nil {
		t.Fatal("Validate command not found")
	}

	if validateCmd.Use != "validate <openapi-spec>" {
		t.Errorf("Expected Use to be 'validate <openapi-spec>', got %q", validateCmd.Use)
	}

	if validateCmd.Short != "Validate an OpenAPI specification" {
		t.Errorf("Expected correct Short description for validate command")
	}
}
