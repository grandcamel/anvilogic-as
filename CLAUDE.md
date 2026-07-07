# CLAUDE.md — anvilogic-as

Minimalist Go CLI for the Anvilogic API, built to be driven by AI agents via
Bash. Companion plugin repo: Anvilogic-Assistant-Skills (doc-only skills).
Full plan + per-wave runbooks: `docs/project-plan.md`. Read the relevant
runbook BEFORE starting wave work.

## Design contract (do not violate)

- **Registry-driven, not command-per-endpoint.** The generic `api` command +
  `api-surface/endpoints.yaml` IS the product. Never hand-write per-endpoint
  subcommands by default (see `docs/wave4-coverage-runbook.md` for the narrow
  exceptions). The predecessor project (jira-as) accumulated ~209 hand-written
  command sites; that is the failure mode this design exists to prevent.
- **Registry upgrade rule:** only raise `confidence`, fill `UNKNOWN` fields,
  append `evidence`. Never delete endpoint ids (deprecate with
  `status: retired`). Never invent paths — every field needs cited evidence.
- **Gated content:** anything from customer-gated docs goes in
  `api-surface/.private/` (gitignored). Public files carry distilled facts
  only (method/path/params). Never name the employer/customer in repo content.
- **Output contract:** stdout = data only. stderr = diagnostics + structured
  error JSON. Exit codes: 0 ok / 1 validation / 2 auth / 3 permission /
  4 not-found / 5 rate-limit / 6 conflict / 7 server. The plugin repo's setup
  command documents this table — keep them in sync.
- **Auth:** `TokenSource` seam in `internal/auth/`. Header scheme is the
  `auth_scheme` config key (default `Bearer` — publicly UNCONFIRMED; Wave 3
  capture confirms or corrects it, then update the default).
- **Mock parity is structural:** `ANVILOGIC_MOCK_MODE=1` swaps the
  `http.RoundTripper` (`internal/mock/`); everything above it is production
  code. Fixtures in `testdata/fixtures/`, keyed
  `<METHOD>_<path_underscored>.json` or a registry `fixture:` key. Fixture
  data must be fully synthetic — never captured tenant data.

## Commands

```sh
make build && make test && make lint      # or: go build ./... etc.
./bin/anvilogic-as registry validate
./bin/anvilogic-as registry list --domain <d> | registry describe <id>
ANVILOGIC_MOCK_MODE=1 ./bin/anvilogic-as api GET /v1/ping   # mock smoke
```

## Gotchas

- **Test exit codes against the compiled binary, not `go run`** — `go run`
  collapses any nonzero child status to 1.
- `go:embed` cannot reference parent dirs; the registry embed lives in root
  `embed.go` (package `anvilogicas`), imported by `internal/registry`.
  Override at runtime with `--registry` / `ANVILOGIC_REGISTRY_PATH`.
- Config cascade (first hit wins): env (`ANVILOGIC_API_KEY`, `ANVILOGIC_BASE_URL`,
  `ANVILOGIC_PROFILE`) > OS keychain > `.claude/settings.local.json` >
  `.claude/settings.json` > `~/.config/anvilogic-as/config.json` > defaults.
- `--paginate` on a raw path requires `--paginate-style`; registry ids with
  `path: UNKNOWN` exit 1 with a hint by design.

## Orchestrated wave execution (Claude Code Workflow tool)

Reference scripts + lessons: `docs/workflows/README.md`. Non-negotiables that
made Waves 1–2 succeed — keep them regardless of which model runs the wave:

1. Bake constants (paths, dates) INTO the workflow script — do not rely on
   the `args` parameter (it has arrived as a JSON string, breaking
   destructuring) and `Date.now()` is unavailable in scripts.
2. Every fan-out agent returns via a StructuredOutput JSON schema; the
   payload must be one valid JSON object, all required fields present.
3. Scaffold/implement agents run their own acceptance commands before
   returning; verification is a separate adversarial pass that re-checks
   evidence/behavior rather than trusting claims, plus a completeness critic;
   bounded fix loop (max 2 rounds) keyed on `severity != minor`.
4. Work in small, independently verifiable steps; when in doubt, verify with
   a real command run rather than reasoning about what code should do.

## Conventions

- Conventional commits (release automation reads them). Configure
  `git config user.name/user.email` to match the account of the remote you
  push to before committing.
- Match existing code style; comments only for non-obvious constraints.
- CI must stay green: build, vet, golangci-lint, `go test -race`, mock smoke.
