// Package install manages the global git-customs pre-push hook.
package install

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// launcher is installed under each hook name in the global hooks dir. It runs
// the customs gate only on pre-push; for every other hook it transparently
// delegates to the repo's own .git/hooks/<name> (in shell, without spawning the
// git-customs binary), so a global core.hooksPath never shadows existing hooks.
const launcher = `#!/bin/sh
hook=$(basename "$0")
if [ "$hook" = "pre-push" ]; then
exec git-customs hook "$@"
fi
prior=$(git config --get customs.priorhookspath 2>/dev/null)
if [ -n "$prior" ]; then
orig="$prior/$hook"
else
gitdir=$(git rev-parse --absolute-git-dir 2>/dev/null) || exit 0
orig="$gitdir/hooks/$hook"
fi
if [ -x "$orig" ]; then
exec "$orig" "$@"
fi
exit 0
`

// standardHooks are the client-side git hooks we install a shim for, so that
// setting a global core.hooksPath does not silently disable a repo's existing
// hooks (lefthook, husky-in-.git/hooks, or plain hooks) for these events.
var standardHooks = []string{
	"applypatch-msg", "pre-applypatch", "post-applypatch",
	"pre-commit", "pre-merge-commit", "prepare-commit-msg", "commit-msg", "post-commit",
	"pre-rebase", "post-checkout", "post-merge", "post-rewrite",
	"pre-push", "pre-auto-gc", "sendemail-validate",
}

const starterConfig = `# git-customs configuration
#
# Each command receives the outgoing diff on stdin. ANY command works — it does
# not have to be AI. The push is allowed only if EVERY command exits 0 AND its
# output is "clear" (empty, or exactly clear_pattern); the first that fails
# blocks the push. Commands are parsed with shell-style quoting, so an argument
# may contain spaces when quoted.
#
# Use 'commands' to combine engines — e.g. a deterministic secret scanner plus
# a best-effort AI check for PII. (Note: an LLM is not a reliable hard gate; it
# is nondeterministic and prompt-injectable. Prefer a deterministic scanner like
# gitleaks for secrets, and treat any AI check as best-effort.)
commands = [
  "gitleaks stdin --no-banner",
  'claude -p "List PII (emails, phone numbers, national IDs, addresses, personal names) in added lines of the diff on stdin. One line per finding. If none, output exactly: OK"',
]

# Single-command shorthand (used only when 'commands' is empty):
# command = 'gitleaks stdin --no-banner'

timeout = "120s"
clear_pattern = "OK"

[repos]
public = true
private = false
`

// Env holds the filesystem/home and a git config runner.
type Env struct {
	Home      string
	GitConfig func(args ...string) ([]byte, error)
}

func (e Env) baseDir() string { return filepath.Join(e.Home, ".config", "git-customs") }

// HooksDir is the directory set as core.hooksPath.
func (e Env) HooksDir() string { return filepath.Join(e.baseDir(), "hooks") }

// priorKey is the git-config key under which we stash the core.hooksPath that
// was in effect before install, so uninstall can restore it. It lives in the
// same scope (--global or --local) as the core.hooksPath we set, which keeps a
// --here install fully reversible and self-contained per repo.
const priorKey = "customs.priorhookspath"

// Install writes the hook and points core.hooksPath at it.
func Install(env Env, local bool, stdout io.Writer) error {
	if err := os.MkdirAll(env.HooksDir(), 0o755); err != nil {
		return err
	}
	for _, h := range standardHooks {
		if err := os.WriteFile(filepath.Join(env.HooksDir(), h), []byte(launcher), 0o755); err != nil {
			return err
		}
	}
	cfgPath := filepath.Join(env.baseDir(), "config.toml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := os.WriteFile(cfgPath, []byte(starterConfig), 0o644); err != nil {
			return err
		}
	}
	scope := "--global"
	if local {
		scope = "--local"
	}
	// Record the prior core.hooksPath in this scope so uninstall can restore it
	// rather than blindly unset (e.g. a repo's lefthook hooksPath under --here).
	// Skip when it is already our own dir: on re-install the current value is
	// ours, and saving it would make uninstall restore our (deleted) dir instead
	// of unsetting. An existing prior key is left intact in that case.
	if out, err := env.GitConfig(scope, "core.hooksPath"); err == nil {
		if prior := strings.TrimSpace(string(out)); prior != "" && prior != env.HooksDir() {
			_, _ = env.GitConfig(scope, priorKey, prior)
		}
	}
	if _, err := env.GitConfig(scope, "core.hooksPath", env.HooksDir()); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "git-customs installed: %s -> core.hooksPath (%s)\n", env.HooksDir(), scope)
	return nil
}

// Uninstall restores prior config and removes the hook dir. It cleans up both
// the global hook and any --here install in the current repo, touching a scope
// only when its core.hooksPath actually points at our dir.
func Uninstall(env Env, stdout io.Writer) error {
	var firstErr error
	for _, scope := range []string{"--global", "--local"} {
		out, err := env.GitConfig(scope, "core.hooksPath")
		if err != nil {
			continue // unset in this scope, or --local outside a repo
		}
		if strings.TrimSpace(string(out)) != env.HooksDir() {
			continue // someone else's hooksPath — leave it alone
		}
		prior, _ := env.GitConfig(scope, priorKey)
		if p := strings.TrimSpace(string(prior)); p != "" {
			if _, err := env.GitConfig(scope, "core.hooksPath", p); err != nil && firstErr == nil {
				firstErr = err
			}
		} else if _, err := env.GitConfig(scope, "--unset-all", "core.hooksPath"); err != nil && firstErr == nil {
			firstErr = err
		}
		_, _ = env.GitConfig(scope, "--unset-all", priorKey)
	}
	_ = os.RemoveAll(env.HooksDir())
	if firstErr != nil {
		return fmt.Errorf("restore core.hooksPath: %w", firstErr)
	}
	_, _ = fmt.Fprintln(stdout, "git-customs uninstalled")
	return nil
}
