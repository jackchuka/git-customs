// Package mapping decides whether git-customs runs for a given repo.
package mapping

import (
	"strings"

	"github.com/jackchuka/git-customs/internal/visibility"
)

// RepoID normalizes a remote URL to "host/owner/repo" (lowercased).
func RepoID(remoteURL string) string {
	host, owner, repo, ok := visibility.ParseRemote(remoteURL)
	if !ok {
		return ""
	}
	return strings.ToLower(host + "/" + owner + "/" + repo)
}

// Resolve decides whether to run for repoID. Explicit entries win over the
// visibility alias; alias() is consulted only when no explicit entry matches.
func Resolve(repos map[string]bool, repoID string, alias func() string) bool {
	if repoID != "" {
		if v, ok := repos[repoID]; ok {
			return v
		}
	}
	if v, ok := repos[alias()]; ok {
		return v
	}
	return false
}
