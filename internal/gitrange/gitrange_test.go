package gitrange

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStdin(t *testing.T) {
	us, err := ParseStdin(strings.NewReader("refs/heads/main aaa refs/heads/main bbb\n"))
	require.NoError(t, err)
	require.Len(t, us, 1)
	assert.Equal(t, "aaa", us[0].LocalSHA)
	assert.Equal(t, "bbb", us[0].RemoteSHA)
}

func TestUpdateClassifiers(t *testing.T) {
	del := Update{LocalSHA: ZeroSHA, RemoteSHA: "bbb"}
	assert.True(t, del.IsDelete())
	assert.False(t, del.IsNewBranch())
	nb := Update{LocalSHA: "aaa", RemoteSHA: ZeroSHA}
	assert.False(t, nb.IsDelete())
	assert.True(t, nb.IsNewBranch())
}

func TestDiffScansCommitPatchesAndSkipsDeletes(t *testing.T) {
	var calls [][]string
	git := func(args ...string) ([]byte, error) {
		calls = append(calls, args)
		return []byte("DIFF\n"), nil
	}
	us := []Update{
		{LocalSHA: "aaa", RemoteSHA: ZeroSHA}, // new branch -> full history
		{LocalSHA: ZeroSHA, RemoteSHA: "bbb"}, // delete -> skipped
		{LocalSHA: "ccc", RemoteSHA: "ddd"},   // normal -> outgoing range
	}
	out, err := Diff(git, us)
	require.NoError(t, err)
	require.Len(t, calls, 2)
	assert.Equal(t, []string{"log", "-p", "--no-color", "aaa"}, calls[0])
	assert.Equal(t, []string{"log", "-p", "--no-color", "ddd..ccc"}, calls[1])
	assert.Contains(t, out, "DIFF")
}
