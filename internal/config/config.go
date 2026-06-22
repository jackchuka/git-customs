// Package config loads git-customs TOML configuration.
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// VisibilityConfig lists hosts/owners always treated as private.
type VisibilityConfig struct {
	SkipHosts  []string `toml:"skip_hosts"`
	SkipOwners []string `toml:"skip_owners"`
}

// Config is the git-customs configuration.
type Config struct {
	Command      string           `toml:"command"`
	Commands     []string         `toml:"commands"`
	Timeout      string           `toml:"timeout"`
	ClearPattern string           `toml:"clear_pattern"`
	Repos        map[string]bool  `toml:"repos"`
	Visibility   VisibilityConfig `toml:"visibility"`
}

// CommandList returns the commands to run, in order. `commands` takes
// precedence over the single `command`; each command receives the full diff
// on stdin and must pass for the push to be allowed.
func (c *Config) CommandList() []string {
	if len(c.Commands) > 0 {
		return c.Commands
	}
	if c.Command != "" {
		return []string{c.Command}
	}
	return nil
}

func defaults() *Config {
	return &Config{Timeout: "60s", ClearPattern: "OK", Repos: map[string]bool{}}
}

// Load reads config from path. A missing file yields defaults and a nil error.
func Load(path string) (*Config, error) {
	c := defaults()
	if _, err := toml.DecodeFile(path, c); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return defaults(), nil
		}
		return nil, err
	}
	if c.ClearPattern == "" {
		c.ClearPattern = "OK"
	}
	if c.Timeout == "" {
		c.Timeout = "60s"
	}
	if c.Repos == nil {
		c.Repos = map[string]bool{}
	}
	return c, nil
}

// TimeoutDuration parses Timeout, defaulting to 60s on error.
func (c *Config) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(c.Timeout)
	if err != nil || d <= 0 {
		return 60 * time.Second
	}
	return d
}

// DefaultPath returns ~/.config/git-customs/config.toml.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "git-customs", "config.toml"), nil
}
