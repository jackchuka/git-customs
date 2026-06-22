// Package visibility resolves whether a git remote is public or private.
package visibility

import (
	"encoding/json"
	"slices"
	"strings"
)

// Visibility is the resolved publicness of a remote.
type Visibility string

const (
	Public  Visibility = "public"
	Private Visibility = "private"
	Unknown Visibility = "unknown"
)

var publicHosts = map[string]bool{
	"github.com": true, "gitlab.com": true, "bitbucket.org": true, "codeberg.org": true,
}

// ParseRemote extracts host/owner/repo from common git remote URL forms.
func ParseRemote(remoteURL string) (host, owner, repo string, ok bool) {
	s := strings.TrimSpace(remoteURL)
	switch {
	case strings.HasPrefix(s, "https://"), strings.HasPrefix(s, "http://"), strings.HasPrefix(s, "ssh://"):
		s = s[strings.Index(s, "://")+3:]
		if at := strings.Index(s, "@"); at != -1 {
			s = s[at+1:]
		}
		slash := strings.Index(s, "/")
		if slash == -1 {
			return "", "", "", false
		}
		host, s = s[:slash], s[slash+1:]
	case strings.Contains(s, "@") && strings.Contains(s, ":"):
		s = s[strings.Index(s, "@")+1:]
		colon := strings.Index(s, ":")
		host, s = s[:colon], s[colon+1:]
	default:
		return "", "", "", false
	}
	s = strings.TrimSuffix(s, ".git")
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", false
	}
	return host, parts[0], parts[1], true
}

// Resolver resolves remote visibility using gh (if present) then host heuristics.
type Resolver struct {
	GH         func(args ...string) ([]byte, error)
	SkipHosts  []string
	SkipOwners []string
}

// Resolve returns the visibility of remoteURL.
func (r Resolver) Resolve(remoteURL string) Visibility {
	host, owner, repo, ok := ParseRemote(remoteURL)
	if !ok {
		return Unknown
	}
	if slices.Contains(r.SkipHosts, host) || slices.Contains(r.SkipOwners, owner) {
		return Private
	}
	if r.GH != nil && host == "github.com" {
		if out, err := r.GH("repo", "view", owner+"/"+repo, "--json", "visibility"); err == nil {
			var v struct {
				Visibility string `json:"visibility"`
			}
			if json.Unmarshal(out, &v) == nil {
				switch strings.ToLower(v.Visibility) {
				case "public":
					return Public
				case "private", "internal":
					return Private
				}
			}
		}
	}
	if publicHosts[host] {
		return Public
	}
	return Unknown
}
