// Package hook orchestrates the git-customs pre-push inspection.
package hook

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/jackchuka/git-customs/internal/chain"
	"github.com/jackchuka/git-customs/internal/config"
	"github.com/jackchuka/git-customs/internal/gitrange"
	"github.com/jackchuka/git-customs/internal/mapping"
	"github.com/jackchuka/git-customs/internal/runner"
	"github.com/jackchuka/git-customs/internal/visibility"
)

// Deps holds the injected dependencies for the hook run.
type Deps struct {
	Config         *config.Config
	Git            func(args ...string) ([]byte, error)
	GH             func(args ...string) ([]byte, error)
	GitDir         string
	PriorHooksPath string
	Exec           runner.ExecFunc
	Bypass         bool
}

// Run performs the pre-push inspection and returns the process exit code.
// hookArgs is the verbatim argument list git passes to a pre-push hook
// (<remote-name> <remote-url>); it is forwarded unchanged to any chained hook.
func Run(d Deps, hookArgs []string, stdinBytes []byte, stdout, stderr io.Writer) int {
	remoteURL := ""
	if len(hookArgs) >= 2 {
		remoteURL = hookArgs[1]
	}
	chainOut := func() int {
		prior := chain.PriorHookPath(d.GitDir, d.PriorHooksPath, "pre-push")
		return chain.RunPrior(prior, hookArgs, stdinBytes, stdout, stderr)
	}

	if d.Bypass {
		_, _ = fmt.Fprintln(stderr, "git-customs: bypassed (CUSTOMS_BYPASS=1)")
		return chainOut()
	}

	updates, err := gitrange.ParseStdin(bytes.NewReader(stdinBytes))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "git-customs: cannot parse push: %v\n", err)
		return 1
	}
	hasWork := false
	for _, u := range updates {
		if !u.IsDelete() {
			hasWork = true
		}
	}
	if !hasWork {
		return chainOut()
	}

	repoID := mapping.RepoID(remoteURL)
	res := visibility.Resolver{
		GH:         d.GH,
		SkipHosts:  d.Config.Visibility.SkipHosts,
		SkipOwners: d.Config.Visibility.SkipOwners,
	}
	alias := func() string { return string(res.Resolve(remoteURL)) }
	if !mapping.Resolve(d.Config.Repos, repoID, alias) {
		return chainOut()
	}

	diff, err := gitrange.Diff(d.Git, updates)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "git-customs: cannot build diff: %v\n", err)
		return 1
	}

	cmds := d.Config.CommandList()
	if len(cmds) == 0 {
		_, _ = fmt.Fprintln(stderr, "git-customs: no command configured")
		return 1 // fail-closed
	}
	target := repoID
	if target == "" {
		target = remoteURL
	}
	_, _ = fmt.Fprintf(stderr, "git-customs: inspecting push to %s (%d %s)\n", target, len(cmds), plural(len(cmds), "check"))
	// Each command receives the full diff on stdin; all must pass to allow.
	for _, cmd := range cmds {
		_, _ = fmt.Fprintf(stderr, "git-customs: → %s\n", commandName(cmd))
		r := runner.Runner{
			Command:      cmd,
			ClearPattern: d.Config.ClearPattern,
			Timeout:      d.Config.TimeoutDuration(),
			Exec:         d.Exec,
		}
		result := r.Run(diff)
		if !r.Allow(result) {
			_, _ = fmt.Fprintf(stderr, "git-customs: push BLOCKED by: %s\n", cmd)
			if result.Stdout != "" {
				_, _ = fmt.Fprintln(stderr, result.Stdout)
			}
			if result.Err != nil {
				_, _ = fmt.Fprintf(stderr, "(command error: %v)\n", result.Err)
			}
			if result.Stderr != "" {
				_, _ = fmt.Fprintln(stderr, result.Stderr)
			}
			_, _ = fmt.Fprintln(stderr, "override: CUSTOMS_BYPASS=1 git push  (or git push --no-verify)")
			return 1
		}
	}
	_, _ = fmt.Fprintf(stderr, "git-customs: ✓ clean — %d %s passed\n", len(cmds), plural(len(cmds), "check"))
	return chainOut()
}

// commandName is the short display label for a configured command: its program
// name, so progress lines stay readable even when the command carries a long
// quoted prompt.
func commandName(cmd string) string {
	if f := strings.Fields(cmd); len(f) > 0 {
		return f[0]
	}
	return cmd
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
