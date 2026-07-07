# Migration Runbook — GitHub → GitLab

Moves both repos to a GitLab namespace with corporate commit identity.
**This public document uses variables** — the operator supplies concrete
values in-session; never commit the target namespace, employer name, or
corporate email to the GitHub copies of these repos.

Variables used below:

| Var | Meaning | Example shape |
|-----|---------|---------------|
| `$GITLAB_NS` | Full GitLab namespace (group/subgroups) | `gitlab.com/<org>/<team>/.../assistant-skills` |
| `$LIBS` | CLI repo project name under `$GITLAB_NS` | `libs` |
| `$SKILLS` | Plugin repo project name under `$GITLAB_NS` | `skills` |
| `$CORP_NAME` / `$CORP_EMAIL` | Corporate git identity | — |

Both existing repos have a **single identity** in history
(`grandcamel <jasonkrue@gmail.com>`), which makes the rewrite trivial.

## Phase A — Create GitLab projects

Blank projects (no README/branch) at `$GITLAB_NS/$LIBS` (this repo) and
`$GITLAB_NS/$SKILLS` (the plugin). Default branch `main`. Note: with nested
subgroups the Go module path becomes the full project path — that's fine, Go
supports it (Phase C).

## Phase B — History identity rewrite (per repo, fresh clone)

```sh
pipx install git-filter-repo   # or pip install --user git-filter-repo
git clone https://github.com/grandcamel/anvilogic-as libs && cd libs
cat > /tmp/mailmap <<EOF
$CORP_NAME <$CORP_EMAIL> grandcamel <jasonkrue@gmail.com>
EOF
git filter-repo --mailmap /tmp/mailmap   # rewrites author + committer
git log --format='%an <%ae> / %cn <%ce>' | sort -u   # must show ONLY the corporate identity
```

Notes: `git filter-repo` removes the origin remote by design (re-add in
Phase D). Message trailers (`Co-Authored-By`, `Claude-Session`) are message
content, untouched by the mailmap; strip them only if policy requires
(`--message-callback`). Set the identity for future commits in each clone:
`git config user.name "$CORP_NAME" && git config user.email "$CORP_EMAIL"`.

## Phase C — Retarget content (before first push)

### C1. Go module path (libs repo — the big one)

```sh
OLD=github.com/grandcamel/anvilogic-as
NEW=$GITLAB_NS/$LIBS            # e.g. gitlab.com/<org>/.../assistant-skills/libs
go mod edit -module "$NEW"
grep -rl "$OLD" --include='*.go' . | xargs sed -i '' "s|$OLD|$NEW|g"
go build ./... && go vet ./... && go test -race ./...
```

Self-imports (root `embed.go` package, any internal cross-imports) are the
only Go references; the grep catches them all. Then fix remaining `$OLD`
references in README.md, docs/, .goreleaser.yaml (see C3), Makefile if any:
`grep -rn "$OLD" . --include='*.md' --include='*.yaml' --include='*.yml'`.

### C2. Cross-repo links + install instructions

In the **skills** repo: `.claude-plugin/plugin.json` and
`marketplace.json` (`repository`, `homepage`), README links, and the setup
command's install path → `go install $NEW/cmd/anvilogic-as@latest`. Because
the GitLab projects are private, document for users:
`export GOPRIVATE=<gitlab-host>/<top-level-group>/*` plus a `~/.netrc` or
`GIT_CONFIG`-based token for `go get`; simplest alternative is installing the
binary from GitLab release artifacts (C3).

### C3. CI and release automation

- `.gitlab-ci.yml` already exists in both repos (added pre-migration) —
  verify the first pipeline is green, then delete `.github/workflows/`.
- **release-please is GitHub-only — remove it on GitLab**: delete
  `release-please-config.json`, `.release-please-manifest.json`, and the
  GitHub release workflow. Replacement (Wave 5): tag manually from
  conventional-commit history and let **goreleaser publish to GitLab
  releases** — in `.goreleaser.yaml` keep the build/archive config, drop the
  `brews:` section (no public tap internally), and run with `GITLAB_TOKEN`
  set (goreleaser auto-detects GitLab via `gitlab_urls` when the remote is
  GitLab). Distribution becomes: GitLab release artifacts + `go install`.
- `LICENSE`: currently MIT with personal copyright. Confirm with the
  employer's policy what internal repos should carry, and update both repos
  consistently.

## Phase D — Push

```sh
git remote add origin git@<gitlab-host>:$GITLAB_NS/$LIBS.git   # or https
git push -u origin main
```

Same for the skills repo. First push to a blank project needs no force.

## Phase E — Verify

1. GitLab pipelines green on both projects.
2. Fresh clone from GitLab: `make build && make test`, mock smoke
   (`ANVILOGIC_MOCK_MODE=1 ./bin/anvilogic-as api GET /v1/ping`), `registry
   validate`, and `git log` shows only the corporate identity.
3. Plugin: manifests point at GitLab; setup command instructions work from a
   clean shell inside the corporate network.
4. `grep -rin "github.com/grandcamel" .` returns nothing in either repo
   (except this runbook's Phase B clone command — update it post-migration).

## Phase F — GitHub disposition

Archive the GitHub repos (Settings → Archive) with a final README note that
development moved (do **not** link the internal location publicly), or delete
them. Decide per employer policy; archiving preserves the public provenance
of the pre-migration history.
