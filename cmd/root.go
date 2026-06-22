// Package cmd implements the git-customs CLI.
package cmd

import (
	"os"

	"github.com/jackchuka/git-customs/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "git-customs",
	Short: "Inspect outgoing pushes to public repos before they leave your machine",
	Long: `git-customs installs one global pre-push hook. Before a push, it decides
from a per-repo mapping whether to inspect, then pipes the outgoing diff to a
configured command (e.g. "claude -p") and blocks the push unless that command
reports clean.`,
	Version:      version.Version,
	SilenceUsage: true,
}

func init() {
	rootCmd.SetVersionTemplate(
		"git-customs {{.Version}} (commit: " + version.Commit + ", built: " + version.BuildDate + ")\n",
	)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
