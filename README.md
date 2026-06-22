# git-customs

**A client-side, cross-host pre-push gate that catches PII and secrets — using
an engine you choose — before anything reaches a public remote.**

git-customs is a *router + pipe*, not a scanner: it decides **which repos** to
inspect (only public-bound pushes, on **any** host — GitHub, GitLab, Gitea,
self-hosted), builds the **diff**, and pipes it to a command **you configure**.
That command can be an LLM (`claude -p "…"`) — which catches free-form **PII**
that regex scanners miss — or a deterministic tool like `gitleaks`. git-customs
itself stays dumb and engine-agnostic.

Pushing to a private repo? It skips silently. Publishing a new public repo? It
scans the whole tree once. Everything in between just works.

## Why it exists (and what it is *not*)

It complements — does not replace — [GitHub push protection](https://docs.github.com/en/code-security/secret-scanning/introduction/about-push-protection).
Push protection is excellent and you should keep it on. But it is **server-side**
(your data already reached GitHub), **GitHub-only**, **secrets-only** (no PII),
and limited to known token types. git-customs covers the seams push protection
leaves:

- **Client-side / pre-upload** — a flagged diff never leaves your machine, not
  even to a server.
- **Cross-host** — works on self-hosted GitLab/Gitea and anywhere else, where
  push protection doesn't exist.
- **PII, not just secrets** — point it at an LLM and it catches names, emails,
  addresses, and IDs that pattern matchers don't.
- **Bring your own engine** — claude, gitleaks, or any executable; not tied to
  one vendor or service.

It is **not** org enforcement: like any local hook it is bypassable
(`--no-verify`) and isn't team-distributed. Treat it as a personal safety net,
with push protection as the server-side backstop.

## Install

Get the binary with Homebrew:

```sh
brew install jackchuka/tap/git-customs
```

Or with Go:

```sh
go install github.com/jackchuka/git-customs@latest
```

Then install the hook:

```sh
git customs install
```

`install` writes a `pre-push` hook to `~/.config/git-customs/hooks/` and points
your **global** `git config core.hooksPath` at it, so it runs in every repo on
the machine. A starter `~/.config/git-customs/config.toml` is created if absent.

Because the binary is named `git-customs` and sits on your `PATH`, git exposes
it as a native subcommand: `git customs install`, `git customs doctor`, etc.

## How a push flows

1. `pre-push` fires → git-customs reads the outgoing ref updates.
2. It resolves whether this repo should be inspected (see mapping below).
   Skipped → the push proceeds untouched.
3. If inspected, it builds the patch of every **outgoing commit** (`git log -p`
   over the pushed range, or the branch's full history on a first push) and
   pipes it to your command on **stdin**. Scanning per-commit — not just the
   endpoint tree diff — catches a secret that was added in one commit and
   removed in a later one but still ships in the pushed history.
4. **Allow only if** the command exits `0` **and** its output is the all-clear
   (empty, or your `clear_pattern`). Otherwise the push is **blocked** and the
   command's output is shown.

This is fail-closed: a missing command, a non-zero exit, or a timeout blocks
the push.

When it inspects, git-customs prints its progress to stderr so you can see it
working (and know which check a slow command is stuck on):

```text
git-customs: inspecting push to github.com/acme/oss (2 checks)
git-customs: → gitleaks
git-customs: → claude
git-customs: ✓ clean — 2 checks passed
```

A skipped push (private repo, branch deletion) stays silent; `CUSTOMS_BYPASS=1`
prints a one-line `bypassed` notice so a skipped gate is never a surprise.

## Configuration — `~/.config/git-customs/config.toml`

Each command receives the diff on stdin. ANY command works — it is not tied to
AI. Commands are parsed with shell-style quoting, so a quoted argument may
contain spaces. The push is allowed only if **every** command exits 0 AND its
output is "clear" (empty, or exactly `clear_pattern`); the first that fails
blocks the push.

```toml
# Combine engines with `commands` — they run in order, all must pass:
commands = [
  "gitleaks stdin --no-banner",                          # deterministic secrets
  'claude -p "List PII (emails, phone numbers, IDs, addresses, names) in added lines of the diff on stdin. One per line. If none, output exactly: OK"',
]
timeout       = "120s"          # per command; exceeding it blocks (fail-closed)
clear_pattern = "OK"            # empty output OR this exact string = clean

[repos]
public  = true                  # run on public repos
private = false                 # skip private repos

# explicit per-repo overrides always win over the public/private alias:
"github.com/jackchuka/blog"  = false   # public, but never inspect
"github.com/acme/secret-oss" = true    # private, but always inspect

[visibility]
skip_hosts  = ["github.example-corp.com"]  # always treated as private
skip_owners = ["my-private-org"]
```

The single-command shorthand still works when you only need one engine:

```toml
command = 'gitleaks stdin --no-banner'   # used only when `commands` is empty
```

### Choosing an engine — reliability matters

git-customs only routes the diff; the engine decides what's a problem. **Use a
deterministic scanner (gitleaks) for the secret gate.** An LLM is *not* a
reliable hard gate: it is nondeterministic (the same diff can pass on one run
and flag on the next) and prompt-injectable (text in the diff can steer its
verdict). Reserve any AI command for **PII**, where no good deterministic tool
exists, and treat it as best-effort rather than a guarantee.

### How visibility is decided

Only when no explicit repo entry matches: git-customs asks `gh` for the true
visibility of GitHub remotes, and falls back to a host heuristic
(github.com / gitlab.com / bitbucket.org / codeberg.org are public) when `gh`
isn't available. Anything it can't confidently classify is treated as private
and skipped.

## Overriding a block

```sh
CUSTOMS_BYPASS=1 git push     # skip inspection for this push
git push --no-verify          # skip all hooks
```

## Coexisting with other hooks

A global `core.hooksPath` makes git read **every** hook type from one directory
and stop looking in `.git/hooks` — which would silently disable a repo's other
hooks. git-customs avoids that: it installs a **shim for every standard hook**.

- **`pre-push`** → runs the customs gate, then chains to the hook git-customs
  shadowed (so an existing pre-push still runs).
- **Every other hook** (`pre-commit`, `commit-msg`, `post-checkout`, …) →
  transparently delegates to the shadowed hook in shell, without even invoking
  git-customs.

The shadowed hook is the one git would otherwise have run: the repo's own
`core.hooksPath` if it had set one (recorded at install time), else
`.git/hooks/<name>`. So **lefthook and husky keep working** — lefthook writes
into `.git/hooks/`, and a `--here` install over husky's `.husky/` chains
through to it. Your pre-commit lint/format hooks fire exactly as before; only
`pre-push` gains the public-gate.

Caveat: a tool that sets its own **repo-local** `core.hooksPath` (husky does)
overrides the global one, so git-customs won't run in that repo. Run
`git customs doctor` to detect it, and `git customs install --here` to install
git-customs for that single repo as well — it records the prior `.husky/` path
and chains every hook through to it.

## Commands

| Command | Purpose |
|---|---|
| `git customs install [--here]` | Install the hook globally (or for the current repo with `--here`) |
| `git customs uninstall` | Remove the hook and restore prior `core.hooksPath` |
| `git customs doctor` | Check the command is on `PATH`, the hook is active, and warn about repo-local bypass |

## License

MIT
