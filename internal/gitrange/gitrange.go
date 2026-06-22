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
func Diff(git func(args ...string) ([]byte, error), updates []Update) (string, error) {
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
			out, err = git("log", "-p", "--no-color", u.LocalSHA)
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
