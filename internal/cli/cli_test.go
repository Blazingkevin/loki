package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CLITestSuite struct {
	suite.Suite
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

func (s *CLITestSuite) TestRootCommand() {
	assert.Equal(s.T(), AppName, rootCmd.Use)
	assert.Equal(s.T(), "Loki: The Trickster for Your APIs", rootCmd.Short)
	assert.Equal(s.T(), Version, rootCmd.Version)

	expectedCommands := []string{"serve", "validate", "version"}
	commands := rootCmd.Commands()
	commandNames := make([]string, len(commands))
	for i, cmd := range commands {
		commandNames[i] = cmd.Name()
	}

	for _, expected := range expectedCommands {
		assert.Contains(s.T(), commandNames, expected, "Expected command %q to be registered", expected)
	}
}

func (s *CLITestSuite) TestVersionCommand() {
	var versionCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "version" {
			versionCmd = cmd
			break
		}
	}

	assert.NotNil(s.T(), versionCmd, "Version command not found")

	if versionCmd != nil {
		assert.Equal(s.T(), "Show version and build information", versionCmd.Short)
		assert.Equal(s.T(), "version", versionCmd.Use)
	}
}

func (s *CLITestSuite) TestServeCommandValidation() {
	var serveCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "serve" {
			serveCmd = cmd
			break
		}
	}

	assert.NotNil(s.T(), serveCmd, "Serve command not found")

	if serveCmd != nil {
		assert.Equal(s.T(), "serve <openapi-spec>", serveCmd.Use)
		assert.Equal(s.T(), "Start a mock API server from OpenAPI specification", serveCmd.Short)

		expectedFlags := []string{"port", "host", "chaos", "profile", "log-level"}
		for _, flagName := range expectedFlags {
			flag := serveCmd.Flags().Lookup(flagName)
			assert.NotNil(s.T(), flag, "Expected flag %q to be defined", flagName)
		}
	}
}

func (s *CLITestSuite) TestValidateCommandValidation() {
	var validateCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "validate" {
			validateCmd = cmd
			break
		}
	}

	assert.NotNil(s.T(), validateCmd, "Validate command not found")

	if validateCmd != nil {
		assert.Equal(s.T(), "validate <openapi-spec>", validateCmd.Use)
		assert.Equal(s.T(), "Validate an OpenAPI specification", validateCmd.Short)
	}
}
