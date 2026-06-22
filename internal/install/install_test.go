package install

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallWritesHookAndSetsGlobalConfig(t *testing.T) {
	home := t.TempDir()
	var cfgArgs [][]string
	env := Env{Home: home, GitConfig: func(args ...string) ([]byte, error) {
		cfgArgs = append(cfgArgs, args)
		return nil, nil
	}}
	require.NoError(t, Install(env, false, &bytes.Buffer{}))

	hook := filepath.Join(home, ".config", "git-customs", "hooks", "pre-push")
	info, err := os.Stat(hook)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&0o111, "hook must be executable")
	body, _ := os.ReadFile(hook)
	assert.Contains(t, string(body), "git-customs hook")

	last := cfgArgs[len(cfgArgs)-1]
	assert.Equal(t, "--global", last[0])
	assert.Equal(t, "core.hooksPath", last[1])

	_, err = os.Stat(filepath.Join(home, ".config", "git-customs", "config.toml"))
	require.NoError(t, err, "starter config must be created")
}

func TestInstallWritesShimsForAllStandardHooks(t *testing.T) {
	home := t.TempDir()
	env := Env{Home: home, GitConfig: func(...string) ([]byte, error) { return nil, nil }}
	require.NoError(t, Install(env, false, &bytes.Buffer{}))

	dir := filepath.Join(home, ".config", "git-customs", "hooks")
	for _, h := range standardHooks {
		info, err := os.Stat(filepath.Join(dir, h))
		require.NoError(t, err, "shim %q must exist", h)
		assert.NotZero(t, info.Mode()&0o111, "shim %q must be executable", h)
	}
	// A non-pre-push shim must delegate to the repo's own hook, not run the gate.
	preCommit, _ := os.ReadFile(filepath.Join(dir, "pre-commit"))
	assert.Contains(t, string(preCommit), `exec "$orig"`, "non-pre-push hooks must pass through")
}

// fakeGitConfig simulates `git config` get/set/unset against an in-memory store
// keyed by "<scope> <name>", so install/uninstall round-trips can be asserted
// end-to-end across both --global and --local scopes.
func fakeGitConfig(store map[string]string) func(args ...string) ([]byte, error) {
	return func(args ...string) ([]byte, error) {
		scope := args[0]
		switch {
		case len(args) == 2: // get: <scope> name
			v, ok := store[scope+" "+args[1]]
			if !ok {
				return nil, fmt.Errorf("not set")
			}
			return []byte(v + "\n"), nil
		case len(args) == 3 && (args[1] == "--unset" || args[1] == "--unset-all"): // unset: <scope> --unset name
			delete(store, scope+" "+args[2])
			return nil, nil
		case len(args) == 3: // set: <scope> name value
			store[scope+" "+args[1]] = args[2]
			return nil, nil
		}
		return nil, nil
	}
}

func TestReinstallThenUninstallClearsHooksPath(t *testing.T) {
	home := t.TempDir()
	store := map[string]string{}
	env := Env{Home: home, GitConfig: fakeGitConfig(store)}

	require.NoError(t, Install(env, false, &bytes.Buffer{}))
	require.NoError(t, Install(env, false, &bytes.Buffer{})) // re-install must not record our own dir as prior
	require.NoError(t, Uninstall(env, &bytes.Buffer{}))

	_, ok := store["--global core.hooksPath"]
	assert.False(t, ok, "uninstall must clear global core.hooksPath, got %q", store["--global core.hooksPath"])
}

func TestUninstallClearsLocalHereInstall(t *testing.T) {
	home := t.TempDir()
	store := map[string]string{}
	env := Env{Home: home, GitConfig: fakeGitConfig(store)}

	require.NoError(t, Install(env, true, &bytes.Buffer{})) // install --here sets a --local hooksPath
	require.Equal(t, env.HooksDir(), store["--local core.hooksPath"])

	require.NoError(t, Uninstall(env, &bytes.Buffer{}))
	_, ok := store["--local core.hooksPath"]
	assert.False(t, ok, "uninstall must clear a --here (local) install, got %q", store["--local core.hooksPath"])
}

func TestHereInstallPreservesAndRestoresPriorLocalHooksPath(t *testing.T) {
	home := t.TempDir()
	store := map[string]string{"--local core.hooksPath": "/repo/.lefthook"}
	env := Env{Home: home, GitConfig: fakeGitConfig(store)}

	require.NoError(t, Install(env, true, &bytes.Buffer{}))
	require.NoError(t, Uninstall(env, &bytes.Buffer{}))

	assert.Equal(t, "/repo/.lefthook", store["--local core.hooksPath"],
		"uninstall must restore the repo's prior local hooksPath, not unset it")
}

func TestUninstallLeavesForeignHooksPathAlone(t *testing.T) {
	home := t.TempDir()
	store := map[string]string{"--global core.hooksPath": "/somewhere/else"}
	env := Env{Home: home, GitConfig: fakeGitConfig(store)}

	require.NoError(t, Uninstall(env, &bytes.Buffer{}))
	assert.Equal(t, "/somewhere/else", store["--global core.hooksPath"],
		"uninstall must not touch a hooksPath that isn't ours")
}

func TestInstallLocalUsesLocalScope(t *testing.T) {
	home := t.TempDir()
	var lastScope string
	env := Env{Home: home, GitConfig: func(args ...string) ([]byte, error) {
		if len(args) >= 2 && args[1] == "core.hooksPath" {
			lastScope = args[0]
		}
		return nil, nil
	}}
	require.NoError(t, Install(env, true, &bytes.Buffer{}))
	assert.Equal(t, "--local", lastScope)
}
