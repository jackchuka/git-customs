package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowCleanOutput(t *testing.T) {
	r := Runner{ClearPattern: "OK"}
	assert.True(t, r.Allow(Result{Stdout: "OK\n", ExitCode: 0}))
	assert.True(t, r.Allow(Result{Stdout: "   ", ExitCode: 0}))
}

func TestBlockOnFindingOrExitOrError(t *testing.T) {
	r := Runner{ClearPattern: "OK"}
	assert.False(t, r.Allow(Result{Stdout: "FOUND aws key", ExitCode: 0}))
	assert.False(t, r.Allow(Result{Stdout: "OK", ExitCode: 3}))
	assert.False(t, r.Allow(Result{Err: context.DeadlineExceeded}))
}

func TestRunPassesDiffAndSplitsCommand(t *testing.T) {
	var gotName, gotStdin string
	var gotArgs []string
	r := Runner{
		Command: "claude -p",
		Timeout: time.Second,
		Exec: func(_ context.Context, name string, args []string, stdin string) Result {
			gotName, gotArgs, gotStdin = name, args, stdin
			return Result{Stdout: "OK"}
		},
	}
	r.Run("THE DIFF")
	assert.Equal(t, "claude", gotName)
	assert.Equal(t, []string{"-p"}, gotArgs)
	assert.Equal(t, "THE DIFF", gotStdin)
}

func TestRunParsesQuotedCommand(t *testing.T) {
	var gotArgs []string
	r := Runner{
		Command: `claude -p "find PII or print OK"`,
		Exec: func(_ context.Context, name string, args []string, stdin string) Result {
			gotArgs = args
			return Result{Stdout: "OK"}
		},
	}
	r.Run("diff")
	require.Equal(t, []string{"-p", "find PII or print OK"}, gotArgs)
}

func TestRunEmptyCommandIsError(t *testing.T) {
	res := Runner{}.Run("x")
	require.Error(t, res.Err)
}

func TestRunUnbalancedQuotesIsError(t *testing.T) {
	res := Runner{Command: `claude -p "unterminated`}.Run("diff")
	require.Error(t, res.Err)
}
