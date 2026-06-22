package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/jackchuka/git-customs/internal/config"
	"github.com/jackchuka/git-customs/internal/gitexec"
	"github.com/jackchuka/git-customs/internal/hook"
	"github.com/jackchuka/git-customs/internal/runner"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:    "hook [remote-name] [remote-url]",
	Short:  "Internal: invoked by the installed pre-push hook",
	Hidden: true,
	Args:   cobra.ArbitraryArgs,
	Run: func(c *cobra.Command, args []string) {
		os.Exit(runHookCmd(args, c.InOrStdin(), c.OutOrStdout(), c.ErrOrStderr()))
	},
}

func init() {
	rootCmd.AddCommand(hookCmd)
}

func runHookCmd(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	stdinBytes, _ := io.ReadAll(stdin)

	cfgPath, err := config.DefaultPath()
	if err != nil {
		return failClosed(stderr, err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return failClosed(stderr, err)
	}
	gitDir, _ := gitexec.GitDir()

	return hook.Run(hook.Deps{
		Config:         cfg,
		Git:            gitexec.Run,
		GH:             gitexec.GH,
		GitDir:         gitDir,
		PriorHooksPath: priorHooksPath(),
		Exec:           runner.DefaultExec,
		Bypass:         os.Getenv("CUSTOMS_BYPASS") == "1",
	}, args, stdinBytes, stdout, stderr)
}

// priorHooksPath returns the core.hooksPath git-customs shadowed at install time
// (e.g. husky's .husky), so the pre-push chain delegates to it rather than to
// .git/hooks. Empty when no custom hooksPath was in effect.
func priorHooksPath() string {
	out, err := gitexec.Run("config", "--get", "customs.priorhookspath")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func failClosed(stderr io.Writer, err error) int {
	_, _ = stderr.Write([]byte("git-customs: " + err.Error() + "\n"))
	return 1
}
