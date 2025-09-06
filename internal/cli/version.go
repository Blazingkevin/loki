package cli

import (
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var (
	buildTime = "unknown"
	gitCommit = "unknown"
	buildBy   = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and build information",
	Long: `Display detailed version and build information for Loki.

This includes the version number, build time, Git commit hash, 
Go version, and system architecture.`,
	Run: runVersion,
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("🔥 %s %s\n", AppName, Version)
	fmt.Printf("   Build time: %s\n", formatBuildTime(buildTime))
	fmt.Printf("   Git commit: %s\n", gitCommit)
	fmt.Printf("   Built by:   %s\n", buildBy)
	fmt.Printf("   Go version: %s\n", runtime.Version())
	fmt.Printf("   Platform:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func formatBuildTime(buildTime string) string {
	if buildTime == "unknown" {
		return "unknown"
	}

	if t, err := time.Parse(time.RFC3339, buildTime); err == nil {
		return t.Format("2025-09-06 12:00:00 MST")
	}

	return buildTime
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
