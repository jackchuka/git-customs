package cmd

import (
	"os"
	"os/exec"
	"strings"

	"github.com/jackchuka/git-customs/internal/config"
	"github.com/jackchuka/git-customs/internal/doctor"
	"github.com/jackchuka/git-customs/internal/gitexec"
	"github.com/jackchuka/git-customs/internal/install"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check command, hooksPath, and per-repo coverage",
	Run: func(c *cobra.Command, _ []string) {
		os.Exit(runDoctorCmd(c))
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctorCmd(c *cobra.Command) int {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		c.PrintErrln("git-customs:", err)
		return 1
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		c.PrintErrln("git-customs:", err)
		return 1
	}
	home, _ := os.UserHomeDir()
	hooksDir := install.Env{Home: home}.HooksDir()
	return doctor.Run(doctor.Deps{
		Config:          cfg,
		HooksDir:        hooksDir,
		GlobalHooksPath: gitConfigValue("--global"),
		LocalHooksPath:  gitConfigValue("--local"),
		LookPath:        exec.LookPath,
	}, c.OutOrStdout())
}

func gitConfigValue(scope string) string {
	out, err := gitexec.Run("config", scope, "core.hooksPath")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
