package doctor

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jackchuka/git-customs/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllGood(t *testing.T) {
	var out bytes.Buffer
	d := Deps{
		Config:   &config.Config{Command: "claude -p"},
		HooksDir: "/h", GlobalHooksPath: "/h",
		LookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
	}
	require.Equal(t, 0, Run(d, &out))
}

func TestMissingCommandFails(t *testing.T) {
	var out bytes.Buffer
	d := Deps{
		Config:   &config.Config{Command: "claude -p"},
		HooksDir: "/h", GlobalHooksPath: "/h",
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
	}
	require.Equal(t, 1, Run(d, &out))
}

func TestChecksEveryCommandInList(t *testing.T) {
	var out bytes.Buffer
	checked := map[string]bool{}
	d := Deps{
		Config:   &config.Config{Commands: []string{"gitleaks stdin", "claude -p x"}},
		HooksDir: "/h", GlobalHooksPath: "/h",
		LookPath: func(name string) (string, error) {
			checked[name] = true
			if name == "claude" {
				return "", errors.New("not found")
			}
			return "/usr/bin/" + name, nil
		},
	}
	require.Equal(t, 1, Run(d, &out)) // claude missing -> fail
	assert.True(t, checked["gitleaks"] && checked["claude"], "both commands checked")
}

func TestWarnsOnLocalOverride(t *testing.T) {
	var out bytes.Buffer
	d := Deps{
		Config:   &config.Config{Command: "claude -p"},
		HooksDir: "/h", GlobalHooksPath: "/h", LocalHooksPath: "/repo/.husky",
		LookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
	}
	Run(d, &out)
	assert.Contains(t, out.String(), "bypass")
}
