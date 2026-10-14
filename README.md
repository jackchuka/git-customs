<div align="center">

# git-customs

**A client-side pre-push gate for secrets and PII, on any git host.**

Inspects only pushes bound for public repos, pipes the outgoing commits<br>
to a scanner you choose, and blocks the push before anything leaves your machine.

[![Test](https://github.com/jackchuka/git-customs/actions/workflows/test.yml/badge.svg)](https://github.com/jackchuka/git-customs/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/jackchuka/git-customs?sort=semver)](https://github.com/jackchuka/git-customs/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

<img src="assets/demo.gif" alt="git-customs blocking a push that contains an AWS key, then allowing it once the key is removed" width="800">

[Installation](#installation) · [Quick Start](#quick-start) · [Configuration](#configuration) · [How It Works](#how-it-works) · [Other Hooks](#coexisting-with-other-hooks)

</div>

---

[GitHub push protection](https://docs.github.com/en/code-security/secret-scanning/introduction/about-push-protection) is excellent and you should keep it on. But it runs **server-side** (your data has already reached GitHub), only on **GitHub**, only for **secrets** (not PII), and only for known token types.

**git-customs** covers those gaps from your side of the wire. It is a router, not a scanner: it decides **which pushes** to inspect, builds the patch of every outgoing commit, and pipes it to a command **you configure**, such as `gitleaks`, or an LLM like `claude -p "…"` for free-form PII that regex scanners miss. Pushes to private repos are skipped silently.

## Features

- **Client-side** — a flagged diff never leaves your machine
- **Public-bound only** — resolves repo visibility via `gh` or the host, with per-repo overrides
- **Any host** — GitHub, GitLab, Bitbucket, Codeberg, and self-hosted remotes via explicit entries
- **Bring your own engine** — any executable that reads stdin: gitleaks, an LLM, your own script
- **Several engines in a row** — every configured command must pass
- **History-aware** — scans each outgoing commit, so a secret added and later removed is still caught
- **Fail-closed** — a missing command, non-zero exit, or timeout blocks the push
- **Plays well with other hooks** — chains to lefthook, husky, or plain `.git/hooks` scripts
- **Native git subcommand** — `git customs install`, `git customs doctor`

## Installation

### Homebrew

```bash
brew install jackchuka/tap/git-customs
```

### Go

```bash
go install github.com/jackchuka/git-customs@latest
```

### Binary

Download a prebuilt binary for Linux, macOS, or Windows from the [releases page](https://github.com/jackchuka/git-customs/releases).

Because the binary is named `git-customs` and sits on your `PATH`, git exposes it as `git customs`.

## Quick Start

1. **Install the hook** for every repo on the machine:

   ```bash
   git customs install
   ```

   This writes hook shims to `~/.config/git-customs/hooks/`, points your **global** `core.hooksPath` at them, and creates a starter `~/.config/git-customs/config.toml` if none exists.

2. **Pick your engines** in `~/.config/git-customs/config.toml`. The starter config runs [gitleaks](https://github.com/gitleaks/gitleaks) and a `claude -p` PII prompt; remove whichever you don't have.

3. **Check the setup:**

   ```bash
   git customs doctor
   ```

4. **Push as usual.** For an inspected push you'll see:

   ```text
   git-customs: inspecting push to github.com/acme/oss (2 checks)
   git-customs: → gitleaks
   git-customs: → claude
   git-customs: ✓ clean — 2 checks passed
   ```

   A blocked push prints the command's output and exits non-zero.

## Commands

| Command                        | Description                                                                                         |
| ------------------------------ | --------------------------------------------------------------------------------------------------- |
| `git customs install`          | Install the hooks globally (`core.hooksPath` in `--global` scope)                                   |
| `git customs install --here`   | Install for the current repo only (`core.hooksPath` in `--local` scope)                             |
| `git customs uninstall`        | Restore the prior `core.hooksPath` in every scope that points at git-customs, and remove the hooks  |
| `git customs doctor`           | Check that each command is on `PATH` and the global hook is active; warn about a repo-local bypass  |
| `git customs --version`        | Print the version                                                                                   |

`uninstall` leaves `config.toml` in place.

### Skipping the gate

```bash
CUSTOMS_BYPASS=1 git push   # skip inspection for this push (prints a one-line notice)
git push --no-verify        # skip all pre-push hooks
```

## Configuration

`~/.config/git-customs/config.toml`:

```toml
# Run in order; every command must pass.
commands = [
  "gitleaks stdin --no-banner",
  'claude -p "List PII (emails, phone numbers, IDs, addresses, names) in added lines of the diff on stdin. One per line. If none, output exactly: OK"',
]
timeout       = "120s"   # per command
clear_pattern = "OK"     # output that counts as clean, besides empty output

[repos]
public  = true           # inspect repos resolved as public
private = false          # skip repos resolved as private

# Explicit entries ("host/owner/repo", lowercase) win over visibility:
"github.com/alice/blog"         = false   # public, but never inspect
"git.example.com/team/sdk"      = true    # self-hosted, always inspect

[visibility]
skip_hosts  = ["github.example.com"]      # always treated as private
skip_owners = ["my-private-org"]
```

| Key                      | Default | Description                                                                                  |
| ------------------------ | ------- | -------------------------------------------------------------------------------------------- |
| `commands`               | —       | Commands to run in order; each gets the full patch on stdin                                  |
| `command`                | —       | Single-command shorthand, used only when `commands` is empty                                 |
| `timeout`                | `60s`   | Per-command timeout as a Go duration; exceeding it blocks the push                           |
| `clear_pattern`          | `OK`    | Output treated as clean, in addition to empty output (compared after trimming whitespace)    |
| `repos.<visibility>`     | —       | `public`, `private`, or `unknown`: whether to inspect repos with that resolved visibility     |
| `repos."<host/owner/repo>"` | —    | Per-repo override; always wins over visibility                                               |
| `visibility.skip_hosts`  | `[]`    | Hosts always treated as private                                                              |
| `visibility.skip_owners` | `[]`    | Owners always treated as private                                                             |

Commands are split with shell-style quoting, so a quoted argument may contain spaces, but they are not run through a shell (no pipes or `$VARS`). A push is allowed only if **every** command exits `0` **and** prints nothing or exactly `clear_pattern`. Without a config file no repo matches, so nothing is inspected; if a repo is inspected but no command is configured, the push is blocked.

> [!TIP]
> Add `-v` (and `--redact`) to the gitleaks command to see each finding in the blocked-push output, not just a count.

### Choosing an engine

git-customs only routes the patch; the engine decides what is a problem. **Use a deterministic scanner such as gitleaks for secrets.** An LLM is not a reliable hard gate: it is nondeterministic (the same diff can pass once and fail the next time) and prompt-injectable (text in the diff can steer its verdict). Use an AI command for **PII**, where no good deterministic tool exists, and treat it as best-effort.

## How It Works

1. git runs the `pre-push` hook, and git-customs reads the outgoing ref updates.
2. It decides whether to inspect the repo:
   - an explicit `repos."host/owner/repo"` entry wins;
   - otherwise it resolves visibility: `skip_hosts` / `skip_owners` → private; for `github.com` remotes, `gh repo view` reports the real visibility; otherwise `github.com`, `gitlab.com`, `bitbucket.org`, and `codeberg.org` count as public; anything else is `unknown`;
   - the matching `repos.public` / `repos.private` / `repos.unknown` value decides, and no match means skip.
3. It builds the patch of every outgoing commit with `git log -p`: the pushed range for an existing branch, or, for a new branch, every commit not already on that remote's tracking refs. Scanning per commit, not just the final tree, catches a secret added in one commit and removed in a later one.
4. It pipes the patch to each command on stdin. The first one that fails blocks the push and shows its output.

Skipped pushes (private repos, branch deletions, nothing new to send) stay silent.

## Coexisting with Other Hooks

A global `core.hooksPath` makes git read **every** hook from one directory and ignore `.git/hooks`, which would silently disable a repo's other hooks. git-customs avoids that by installing a shim for every standard client-side hook:

- **`pre-push`** runs the customs gate, then chains to the hook it shadowed.
- **Every other hook** (`pre-commit`, `commit-msg`, `post-checkout`, …) delegates straight to the shadowed hook in shell, without starting git-customs.

The shadowed hook is the one git would otherwise have run: the prior `core.hooksPath` recorded at install time, or `.git/hooks/<name>`. So lefthook (which writes to `.git/hooks/`) keeps working unchanged.

A tool that sets its own **repo-local** `core.hooksPath`, as husky does, overrides the global one, so git-customs won't run in that repo. `git customs doctor` detects this; `git customs install --here` installs git-customs for that repo too, recording the prior `.husky/` path and chaining every hook through to it.

## Limitations

git-customs is a personal safety net, not org enforcement. Like any local hook it can be bypassed with `--no-verify` and isn't distributed to a team. Keep push protection on as the server-side backstop.

## Development

```bash
go test ./...
go build -o git-customs .
vhs assets/demo.tape   # re-render the demo (needs vhs and gitleaks)
```

## License

[MIT](LICENSE)
