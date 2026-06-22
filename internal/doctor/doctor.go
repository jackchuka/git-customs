// Package doctor reports git-customs configuration and coverage health.
package doctor

import (
	"fmt"
	"io"
	"strings"

	"github.com/jackchuka/git-customs/internal/config"
)

// Deps holds the inputs for a doctor run.
type Deps struct {
	Config          *config.Config
	HooksDir        string
	GlobalHooksPath string
	LocalHooksPath  string
	LookPath        func(string) (string, error)
}

// Run prints checks and returns 0 when critical checks pass, else 1.
func Run(d Deps, stdout io.Writer) int {
	ok := true

	cmds := d.Config.CommandList()
	if len(cmds) == 0 {
		_, _ = fmt.Fprintln(stdout, "✗ no command configured")
		ok = false
	}
	for _, cmd := range cmds {
		name := ""
		if f := strings.Fields(cmd); len(f) > 0 {
			name = f[0]
		}
		if _, err := d.LookPath(name); err != nil {
			_, _ = fmt.Fprintf(stdout, "✗ command %q not found on PATH\n", name)
			ok = false
		} else {
			_, _ = fmt.Fprintf(stdout, "✓ command: %s\n", name)
		}
	}

	if d.HooksDir != "" && d.GlobalHooksPath == d.HooksDir {
		_, _ = fmt.Fprintln(stdout, "✓ global core.hooksPath active")
	} else {
		_, _ = fmt.Fprintf(stdout, "✗ global core.hooksPath is %q, expected %q (run: git customs install)\n",
			d.GlobalHooksPath, d.HooksDir)
		ok = false
	}

	if d.LocalHooksPath != "" && d.LocalHooksPath != d.HooksDir {
		_, _ = fmt.Fprintf(stdout, "⚠ this repo sets a local core.hooksPath (%q) — customs will bypass it here; run: git customs install --here\n",
			d.LocalHooksPath)
	}

	if ok {
		return 0
	}
	return 1
}
