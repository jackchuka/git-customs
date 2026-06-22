package mapping

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepoID(t *testing.T) {
	require.Equal(t, "github.com/jackchuka/blog", RepoID("git@github.com:JackChuka/Blog.git"))
}

func TestExplicitEntryWinsAndSkipsAlias(t *testing.T) {
	aliasCalled := false
	alias := func() string { aliasCalled = true; return "public" }
	repos := map[string]bool{"public": true, "github.com/jackchuka/blog": false}
	require.False(t, Resolve(repos, "github.com/jackchuka/blog", alias))
	require.False(t, aliasCalled, "alias must not be called when explicit entry matches")
}

func TestFallsBackToAlias(t *testing.T) {
	repos := map[string]bool{"public": true, "private": false}
	require.True(t, Resolve(repos, "github.com/x/y", func() string { return "public" }))
	require.False(t, Resolve(repos, "github.com/x/y", func() string { return "private" }))
}

func TestUnknownAliasSkips(t *testing.T) {
	repos := map[string]bool{"public": true}
	require.False(t, Resolve(repos, "github.com/x/y", func() string { return "unknown" }))
}
