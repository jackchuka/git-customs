// Package gitexec wraps the real git and gh binaries.
package gitexec

import (
	"os/exec"
	"strings"
)

// Run executes git with args and returns stdout.
func Run(args ...string) ([]byte, error) { return exec.Command("git", args...).Output() }

// GH executes gh with args and returns stdout.
func GH(args ...string) ([]byte, error) { return exec.Command("gh", args...).Output() }

// GitDir returns the absolute .git directory for the current repo.
func GitDir() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--absolute-git-dir").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
