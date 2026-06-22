// Package chain delegates to a repo's pre-existing hook shadowed by the global hooksPath.
package chain

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// PriorHookPath returns the hook shadowed when git-customs took over
// core.hooksPath, if it exists and is executable. That is the hook of the same
// name under the prior core.hooksPath saved at install time (e.g. husky's
// .husky), or the repo's .git/hooks when no custom hooksPath was in effect. A
// relative priorHooksPath (as husky configures) is resolved against the current
// working directory, which git sets to the top of the working tree when running
// a hook — the same base git itself uses for a relative core.hooksPath.
func PriorHookPath(gitDir, priorHooksPath, hookName string) string {
	dir := priorHooksPath
	if dir == "" {
		dir = filepath.Join(gitDir, "hooks")
	}
	p := filepath.Join(dir, hookName)
	info, err := os.Stat(p)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return ""
	}
	return p
}

// RunPrior executes the prior hook, forwarding stdin/args. Empty path → allow (0).
func RunPrior(path string, args []string, stdin []byte, stdout, stderr io.Writer) int {
	if path == "" {
		return 0
	}
	cmd := exec.Command(path, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 1
}
