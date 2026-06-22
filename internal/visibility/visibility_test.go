package visibility

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRemote(t *testing.T) {
	cases := []struct{ in, host, owner, repo string }{
		{"https://github.com/jackchuka/blog.git", "github.com", "jackchuka", "blog"},
		{"git@github.com:jackchuka/blog.git", "github.com", "jackchuka", "blog"},
		{"ssh://git@ghe.corp.com/team/svc", "ghe.corp.com", "team", "svc"},
	}
	for _, c := range cases {
		h, o, r, ok := ParseRemote(c.in)
		require.True(t, ok, c.in)
		assert.Equal(t, c.host, h)
		assert.Equal(t, c.owner, o)
		assert.Equal(t, c.repo, r)
	}
}

func TestResolveSkipOwnerIsPrivate(t *testing.T) {
	r := Resolver{SkipOwners: []string{"secret-org"}}
	require.Equal(t, Private, r.Resolve("https://github.com/secret-org/x.git"))
}

func TestResolveUsesGHWhenAvailable(t *testing.T) {
	r := Resolver{GH: func(args ...string) ([]byte, error) {
		return []byte(`{"visibility":"private"}`), nil
	}}
	require.Equal(t, Private, r.Resolve("https://github.com/jackchuka/blog.git"))
}

func TestResolveFallsBackToPublicHostWhenGHFails(t *testing.T) {
	r := Resolver{GH: func(args ...string) ([]byte, error) { return nil, errors.New("no gh") }}
	require.Equal(t, Public, r.Resolve("git@github.com:jackchuka/blog.git"))
}

func TestResolveUnknownHost(t *testing.T) {
	require.Equal(t, Unknown, Resolver{}.Resolve("git@ghe.corp.com:team/svc.git"))
}
