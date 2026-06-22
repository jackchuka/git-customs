// Package runner executes the configured inspection command and decides allow/block.
package runner

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/google/shlex"
)

// Result captures a command invocation outcome.
type Result struct {
	Stdout, Stderr string
	ExitCode       int
	Err            error
}

// ExecFunc runs a command with stdin and returns its Result.
type ExecFunc func(ctx context.Context, name string, args []string, stdin string) Result

// Runner runs the configured command and applies the decision contract.
type Runner struct {
	Command      string
	ClearPattern string
	Timeout      time.Duration
	Exec         ExecFunc
}

var errNoCommand = errors.New("no command configured")

// Run executes the command with diff on stdin under the configured timeout.
// Command is parsed with shell-style quoting, so an argument may contain
// spaces when quoted (e.g. `claude -p "scan this diff"`).
func (r Runner) Run(diff string) Result {
	fields, err := shlex.Split(r.Command)
	if err != nil {
		return Result{ExitCode: 1, Err: err}
	}
	if len(fields) == 0 {
		return Result{ExitCode: 1, Err: errNoCommand}
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	run := r.Exec
	if run == nil {
		run = DefaultExec
	}
	return run(ctx, fields[0], fields[1:], diff)
}

// Allow returns true only when the command exited 0 with clear output.
func (r Runner) Allow(res Result) bool {
	if res.Err != nil || res.ExitCode != 0 {
		return false
	}
	out := strings.TrimSpace(res.Stdout)
	return out == "" || out == r.ClearPattern
}

// DefaultExec runs the command for real, with diff on stdin.
func DefaultExec(ctx context.Context, name string, args []string, stdin string) Result {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errBuf
	err := cmd.Run()
	res := Result{Stdout: out.String(), Stderr: errBuf.String()}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
		} else {
			res.Err = err
			res.ExitCode = 1
		}
	}
	return res
}
