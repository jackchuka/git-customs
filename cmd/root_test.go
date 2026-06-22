package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCommandUse(t *testing.T) {
	require.Equal(t, "git-customs", rootCmd.Use)
}

func TestRootHasSubcommands(t *testing.T) {
	// install, uninstall, doctor, hook are registered by their files' init().
	names := map[string]bool{}
	for _, c := range rootCmd.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{"install", "uninstall", "doctor", "hook"} {
		require.True(t, names[want], "missing subcommand %q", want)
	}
}
