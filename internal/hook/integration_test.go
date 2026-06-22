package hook

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jackchuka/git-customs/internal/config"
	"github.com/jackchuka/git-customs/internal/runner"
	"github.com/stretchr/testify/require"
)

func gitInitRepo(t *testing.T) (dir, gitDir, headSHA string) {
	t.Helper()
	dir = t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	run("init", "-q")
	run("config", "user.email", "t@example.com")
	run("config", "user.name", "t")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hello\n"), 0o644))
	run("add", ".")
	run("commit", "-q", "-m", "init")
	gd, _ := exec.Command("git", "-C", dir, "rev-parse", "--absolute-git-dir").Output()
	sha, _ := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	return dir, string(bytes.TrimSpace(gd)), string(bytes.TrimSpace(sha))
}

func TestIntegrationBlockThenAllow(t *testing.T) {
	dir, gitDir, head := gitInitRepo(t)
	gitFromRepo := func(args ...string) ([]byte, error) {
		return exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	}
	cfg := &config.Config{Command: "stub", ClearPattern: "OK", Repos: map[string]bool{"public": true}}
	stdin := []byte("refs/heads/main " + head + " refs/heads/main " + "0000000000000000000000000000000000000000\n")

	d := Deps{
		Config: cfg, Git: gitFromRepo, GitDir: gitDir,
		GH: func(args ...string) ([]byte, error) { return []byte(`{"visibility":"public"}`), nil },
		Exec: func(context.Context, string, []string, string) runner.Result {
			return runner.Result{Stdout: "FOUND key"}
		},
	}
	args := []string{"origin", "git@github.com:jackchuka/x.git"}
	var out, errOut bytes.Buffer
	require.Equal(t, 1, Run(d, args, stdin, &out, &errOut), errOut.String())

	d.Exec = func(context.Context, string, []string, string) runner.Result { return runner.Result{Stdout: "OK"} }
	out.Reset()
	errOut.Reset()
	require.Equal(t, 0, Run(d, args, stdin, &out, &errOut), errOut.String())
}

// A passing gate must chain to the repo's prior pre-push hook, forwarding git's
// verbatim <remote-name> <remote-url> args and the original stdin.
func TestIntegrationChainsPriorHookWithOriginalArgs(t *testing.T) {
	dir, gitDir, head := gitInitRepo(t)
	gitFromRepo := func(args ...string) ([]byte, error) {
		return exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	}
	// A prior pre-push hook that records the args and stdin it was handed.
	argsFile := filepath.Join(dir, "prior-args.txt")
	hookPath := filepath.Join(gitDir, "hooks", "pre-push")
	require.NoError(t, os.MkdirAll(filepath.Dir(hookPath), 0o755))
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\ncat >> " + argsFile + "\nexit 0\n"
	require.NoError(t, os.WriteFile(hookPath, []byte(script), 0o755))

	stdin := []byte("refs/heads/main " + head + " refs/heads/main " + "0000000000000000000000000000000000000000\n")
	d := Deps{
		Config: &config.Config{Command: "stub", ClearPattern: "OK", Repos: map[string]bool{"public": true}},
		Git:    gitFromRepo, GitDir: gitDir,
		GH:   func(...string) ([]byte, error) { return []byte(`{"visibility":"public"}`), nil },
		Exec: func(context.Context, string, []string, string) runner.Result { return runner.Result{Stdout: "OK"} },
	}
	args := []string{"origin", "https://github.com/jackchuka/x.git"}
	var out, errOut bytes.Buffer
	require.Equal(t, 0, Run(d, args, stdin, &out, &errOut), errOut.String())

	got, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	want := "origin\nhttps://github.com/jackchuka/x.git\n" + string(stdin)
	require.Equal(t, want, string(got))
}
