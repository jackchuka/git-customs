package cmd

import (
	"fmt"
	"os"

	"github.com/jackchuka/git-customs/internal/gitexec"
	"github.com/jackchuka/git-customs/internal/install"
	"github.com/spf13/cobra"
)

var flagInstallHere bool

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the global pre-push hook (use --here for the current repo only)",
	RunE: func(c *cobra.Command, _ []string) error {
		env, err := realEnv()
		if err != nil {
			return err
		}
		return install.Install(env, flagInstallHere, c.OutOrStdout())
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the global pre-push hook and restore prior config",
	RunE: func(c *cobra.Command, _ []string) error {
		env, err := realEnv()
		if err != nil {
			return err
		}
		return install.Uninstall(env, c.OutOrStdout())
	},
}

func init() {
	installCmd.Flags().BoolVar(&flagInstallHere, "here", false, "Install for the current repo only (repo-local core.hooksPath)")
	rootCmd.AddCommand(installCmd, uninstallCmd)
}

func realEnv() (install.Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return install.Env{}, fmt.Errorf("home dir: %w", err)
	}
	return install.Env{
		Home:      home,
		GitConfig: func(args ...string) ([]byte, error) { return gitexec.Run(append([]string{"config"}, args...)...) },
	}, nil
}
