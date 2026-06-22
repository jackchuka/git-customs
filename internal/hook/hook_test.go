package hook

import (
	"bytes"
	"context"
	"testing"

	"github.com/jackchuka/git-customs/internal/config"
	"github.com/jackchuka/git-customs/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseDeps() Deps {
	return Deps{
		Config: &config.Config{
			Command: "fake", ClearPattern: "OK",
			Repos: map[string]bool{"public": true, "private": false},
		},
		Git:    func(args ...string) ([]byte, error) { return []byte("DIFF"), nil },
		GH:     func(args ...string) ([]byte, error) { return []byte(`{"visibility":"public"}`), nil },
		GitDir: "/nonexistent", // no prior hook -> chain is a no-op
		Exec:   func(context.Context, string, []string, string) runner.Result { return runner.Result{Stdout: "OK"} },
	}
}

const pushLine = "refs/heads/main aaa refs/heads/main bbb\n"

func TestAllowsCleanPublic(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run(baseDeps(), []string{"origin", "git@github.com:jackchuka/blog.git"}, []byte(pushLine), &out, &errOut)
	require.Equal(t, 0, code, errOut.String())
}

func TestBlocksOnFinding(t *testing.T) {
	d := baseDeps()
	d.Exec = func(context.Context, string, []string, string) runner.Result {
		return runner.Result{Stdout: "FOUND secret in config"}
	}
	var out, errOut bytes.Buffer
	code := Run(d, []string{"origin", "git@github.com:jackchuka/blog.git"}, []byte(pushLine), &out, &errOut)
	require.Equal(t, 1, code)
	assert.Contains(t, errOut.String(), "FOUND secret")
}

func TestSkipsPrivateWithoutRunningCommand(t *testing.T) {
	d := baseDeps()
	d.GH = func(args ...string) ([]byte, error) { return []byte(`{"visibility":"private"}`), nil }
	d.Exec = func(context.Context, string, []string, string) runner.Result {
		t.Fatal("command must not run for skipped repo")
		return runner.Result{}
	}
	var out, errOut bytes.Buffer
	require.Equal(t, 0, Run(d, []string{"origin", "git@github.com:jackchuka/blog.git"}, []byte(pushLine), &out, &errOut))
	// A skipped (private) repo must stay silent — the README promises this.
	assert.Empty(t, errOut.String(), "skipped push must not print an indicator")
}

func TestPrintsRunningIndicatorWhenInspecting(t *testing.T) {
	var out, errOut bytes.Buffer
	require.Equal(t, 0, Run(baseDeps(), []string{"origin", "git@github.com:jackchuka/blog.git"}, []byte(pushLine), &out, &errOut))
	s := errOut.String()
	assert.Contains(t, s, "inspecting push to github.com/jackchuka/blog")
	assert.Contains(t, s, "→ fake") // the configured command's program name
	assert.Contains(t, s, "✓ clean")
}

func TestBypassPrintsIndicator(t *testing.T) {
	d := baseDeps()
	d.Bypass = true
	var out, errOut bytes.Buffer
	require.Equal(t, 0, Run(d, []string{"origin", "git@github.com:jackchuka/blog.git"}, []byte(pushLine), &out, &errOut))
	assert.Contains(t, errOut.String(), "bypassed")
}

func TestMultipleCommandsBlockIfAnyFlags(t *testing.T) {
	d := baseDeps()
	d.Config.Command = ""
	d.Config.Commands = []string{"gitleaks", "claude -p x"}
	d.Exec = func(_ context.Context, name string, _ []string, _ string) runner.Result {
		if name == "claude" {
			return runner.Result{Stdout: "seed.json:2 PII: a@b.com"}
		}
		return runner.Result{Stdout: "OK"} // gitleaks clean
	}
	var out, errOut bytes.Buffer
	code := Run(d, []string{"origin", "git@github.com:jackchuka/blog.git"}, []byte(pushLine), &out, &errOut)
	require.Equal(t, 1, code)
	assert.Contains(t, errOut.String(), "PII: a@b.com")
	assert.Contains(t, errOut.String(), "BLOCKED by: claude -p x")
}

func TestMultipleCommandsAllowIfAllPass(t *testing.T) {
	d := baseDeps()
	d.Config.Command = ""
	d.Config.Commands = []string{"gitleaks", "claude -p x"}
	d.Exec = func(context.Context, string, []string, string) runner.Result {
		return runner.Result{Stdout: "OK"}
	}
	var out, errOut bytes.Buffer
	require.Equal(t, 0, Run(d, []string{"origin", "git@github.com:jackchuka/blog.git"}, []byte(pushLine), &out, &errOut), errOut.String())
}

func TestBypassSkipsCommand(t *testing.T) {
	d := baseDeps()
	d.Bypass = true
	d.Exec = func(context.Context, string, []string, string) runner.Result {
		t.Fatal("bypass must skip command")
		return runner.Result{}
	}
	var out, errOut bytes.Buffer
	require.Equal(t, 0, Run(d, []string{"origin", "git@github.com:jackchuka/blog.git"}, []byte(pushLine), &out, &errOut))
}
