package chain

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriorHookPathDetectsExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable-bit semantics differ on windows")
	}
	dir := t.TempDir()
	hooks := filepath.Join(dir, "hooks")
	require.NoError(t, os.MkdirAll(hooks, 0o755))
	p := filepath.Join(hooks, "pre-push")
	require.NoError(t, os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	assert.Equal(t, p, PriorHookPath(dir, "", "pre-push"))
	assert.Equal(t, "", PriorHookPath(dir, "", "pre-commit"))
}

func TestPriorHookPathPrefersSavedHooksPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable-bit semantics differ on windows")
	}
	gitDir := t.TempDir()
	// A repo with a husky-style custom hooksPath: the prior hook lives there,
	// not under .git/hooks, so chaining must follow the saved path.
	prior := t.TempDir()
	p := filepath.Join(prior, "pre-push")
	require.NoError(t, os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	assert.Equal(t, p, PriorHookPath(gitDir, prior, "pre-push"))
	assert.Equal(t, "", PriorHookPath(gitDir, prior, "pre-commit"))
}

func TestRunPriorPropagatesExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script hook")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "pre-push")
	require.NoError(t, os.WriteFile(p, []byte("#!/bin/sh\nexit 7\n"), 0o755))
	var out, errOut bytes.Buffer
	assert.Equal(t, 7, RunPrior(p, nil, []byte(""), &out, &errOut))
}

func TestRunPriorEmptyPathAllows(t *testing.T) {
	var out, errOut bytes.Buffer
	assert.Equal(t, 0, RunPrior("", nil, nil, &out, &errOut))
}
