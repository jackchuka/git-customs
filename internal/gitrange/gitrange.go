// Package gitrange parses git pre-push input and builds the diff to inspect.
package gitrange

import (
	"bufio"
	"io"
	"strings"
)

const ZeroSHA = "0000000000000000000000000000000000000000"

// Update is one ref update line from git's pre-push stdin.
type Update struct {
	LocalRef, LocalSHA, RemoteRef, RemoteSHA string
}

func isZero(sha string) bool { return sha == "" || strings.Trim(sha, "0") == "" }

// IsDelete reports a branch deletion (local side is zero).
func (u Update) IsDelete() bool { return isZero(u.LocalSHA) }

// IsNewBranch reports a first push of a branch (remote side is zero).
func (u Update) IsNewBranch() bool { return !u.IsDelete() && isZero(u.RemoteSHA) }

// ParseStdin reads "<local ref> <local sha> <remote ref> <remote sha>" lines.
func ParseStdin(r io.Reader) ([]Update, error) {
	var us []Update
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) != 4 {
			continue
		}
		us = append(us, Update{f[0], f[1], f[2], f[3]})
	}
	return us, sc.Err()
}

// Diff returns the content to inspect for every non-delete update: the patch of
// each outgoing commit, not the endpoint tree delta. This matters because a
// secret added in one commit and removed in a later one leaves no trace in the
// final tree diff, yet both commits are pushed and the secret lives on in remote
// history. `git log -p` emits every commit's patch (and message) so the scanner
// sees content that an endpoint diff would hide.
//
// remote is the remote name git passed to the hook. A new branch has no remote
// SHA to bound the range, so outgoing commits are found by excluding everything
// already reachable from that remote's tracking refs. Without that exclusion the
// walk covers the branch's entire ancestry -- including all of main -- and any
// secret anywhere in history blocks every new branch forever, which trains
// people to bypass the scanner rather than fix the leak.
//
// The exclusion relies on local remote-tracking refs. A stale or single-branch
// clone excludes less than it could and over-scans, which fails toward a false
// positive rather than a miss. The reverse direction is a real gap: `git fetch`
// does not prune by default, so a tracking ref for a branch since deleted from
// the remote still excludes commits the remote no longer has.
func Diff(git func(args ...string) ([]byte, error), updates []Update, remote string) (string, error) {
	var b strings.Builder
	for _, u := range updates {
		if u.IsDelete() {
			continue
		}
		var (
			out []byte
			err error
		)
		if u.IsNewBranch() {
			out, err = git(append([]string{"log", "-p", "--no-color", u.LocalSHA}, excludeRemote(remote)...)...)
		} else {
			out, err = git("log", "-p", "--no-color", u.RemoteSHA+".."+u.LocalSHA)
		}
		if err != nil {
			return "", err
		}
		b.Write(out)
	}
	return b.String(), nil
}

// excludeRemote builds the "--not --remotes=<name>" arguments that drop commits
// the named remote already has. Git passes a remote name, but a push straight to
// a URL passes that URL instead, which no tracking ref is named after. Nothing
// local records what that destination already holds, so no exclusion is applied
// and the walk over-scans. Bare --remotes would be the opposite of safe here: it
// excludes everything reachable from every tracking ref, so pushing an
// already-fetched branch to a fresh URL would leave nothing to scan at all.
func excludeRemote(remote string) []string {
	if remote == "" || strings.ContainsAny(remote, "/:") {
		return nil
	}
	return []string{"--not", "--remotes=" + remote}
}
