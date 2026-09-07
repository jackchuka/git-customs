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
		{LocalSHA: "aaa", RemoteSHA: ZeroSHA}, // new branch -> outgoing only
		{LocalSHA: ZeroSHA, RemoteSHA: "bbb"}, // delete -> skipped
		{LocalSHA: "ccc", RemoteSHA: "ddd"},   // normal -> outgoing range
	}
	out, err := Diff(git, us, "origin")
	require.NoError(t, err)
	require.Len(t, calls, 2)
	assert.Equal(t, []string{"log", "-p", "--no-color", "aaa", "--not", "--remotes=origin"}, calls[0])
	assert.Equal(t, []string{"log", "-p", "--no-color", "ddd..ccc"}, calls[1])
	assert.Contains(t, out, "DIFF")
}

// A new branch must not re-scan history the remote already has. Walking the
// branch's full ancestry means one secret anywhere in the past blocks every new
// branch permanently, leaving CUSTOMS_BYPASS=1 as the only way to work.
func TestDiffNewBranchExcludesCommitsAlreadyOnRemote(t *testing.T) {
	var calls [][]string
	git := func(args ...string) ([]byte, error) {
		calls = append(calls, args)
		return nil, nil
	}
	_, err := Diff(git, []Update{{LocalSHA: "aaa", RemoteSHA: ZeroSHA}}, "origin")
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Equal(t, []string{"log", "-p", "--no-color", "aaa", "--not", "--remotes=origin"}, calls[0],
		"new branch must scan only outgoing commits, not its entire ancestry")
}

func TestDiffNewBranchRemoteNameVariants(t *testing.T) {
	cases := []struct {
		name, remote string
		want         []string
	}{
		{"named remote", "upstream", []string{"--not", "--remotes=upstream"}},
		// No tracking ref is named after a URL, so nothing local says what that
		// destination already has. Over-scan instead of excluding blindly: bare
		// --remotes would drop every fetched commit and scan nothing at all.
		{"empty remote", "", nil},
		{"ssh url", "git@github.com:o/r.git", nil},
		{"https url", "https://github.com/o/r.git", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls [][]string
			git := func(args ...string) ([]byte, error) {
				calls = append(calls, args)
				return nil, nil
			}
			_, err := Diff(git, []Update{{LocalSHA: "aaa", RemoteSHA: ZeroSHA}}, tc.remote)
			require.NoError(t, err)
			require.Len(t, calls, 1)
			assert.Equal(t, append([]string{"log", "-p", "--no-color", "aaa"}, tc.want...), calls[0])
		})
	}
}

// An existing branch is already bounded by the remote SHA, so it must stay a
// plain range -- adding an exclusion there would be redundant.
func TestDiffExistingBranchIgnoresRemoteName(t *testing.T) {
	var calls [][]string
	git := func(args ...string) ([]byte, error) {
		calls = append(calls, args)
		return nil, nil
	}
	_, err := Diff(git, []Update{{LocalSHA: "ccc", RemoteSHA: "ddd"}}, "origin")
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Equal(t, []string{"log", "-p", "--no-color", "ddd..ccc"}, calls[0])
}
