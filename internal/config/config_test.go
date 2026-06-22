package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
	return p
}

func TestLoadParsesReposAndVisibility(t *testing.T) {
	p := writeTemp(t, `
command = "claude -p"
timeout = "30s"
[repos]
public = true
private = false
"github.com/jackchuka/blog" = false
[visibility]
skip_hosts = ["ghe.corp.com"]
skip_owners = ["secret-org"]
`)
	c, err := Load(p)
	require.NoError(t, err)
	require.Equal(t, "claude -p", c.Command)
	require.Equal(t, true, c.Repos["public"])
	require.Equal(t, false, c.Repos["github.com/jackchuka/blog"])
	require.Equal(t, 30*time.Second, c.TimeoutDuration())
	require.Equal(t, []string{"ghe.corp.com"}, c.Visibility.SkipHosts)
	require.Equal(t, []string{"secret-org"}, c.Visibility.SkipOwners)
}

func TestLoadParsesCommandsList(t *testing.T) {
	p := writeTemp(t, `commands = ["gitleaks stdin", "claude -p x"]`)
	c, err := Load(p)
	require.NoError(t, err)
	require.Equal(t, []string{"gitleaks stdin", "claude -p x"}, c.Commands)
}

func TestCommandListPrecedence(t *testing.T) {
	require.Equal(t, []string{"a", "b"}, (&Config{Command: "single", Commands: []string{"a", "b"}}).CommandList())
	require.Equal(t, []string{"single"}, (&Config{Command: "single"}).CommandList())
	require.Nil(t, (&Config{}).CommandList())
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	require.NoError(t, err)
	require.Equal(t, "OK", c.ClearPattern)
	require.Equal(t, 60*time.Second, c.TimeoutDuration())
	require.NotNil(t, c.Repos)
}
