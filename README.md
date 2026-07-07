# anvilogic-as

A minimalist, **agent-first** CLI for the Anvilogic SaaS control-plane API.

The CLI is **registry-driven**: everything it knows about the API surface
lives in [`api-surface/endpoints.yaml`](api-surface/endpoints.yaml), a
knowledge base synthesized from public sources and embedded into the binary
at build time. Commands resolve endpoint ids against that registry, inject
authentication transparently, and keep a strict output contract so both
humans and automation (LLM agents, scripts, SOAR pipelines) can drive it:

- **stdout carries data only** (`--output json|ndjson|text`, default `json`).
- **stderr carries diagnostics** plus one structured error object:
  `{"error":{"code":"...","status":N,"hint":"..."}}`.
- **Exit codes encode the failure class**: `0` ok, `1` validation, `2` auth,
  `3` permission, `4` not-found, `5` rate-limit-exhausted, `6` conflict,
  `7` server.

## Quick start

```sh
# Build
make build            # bin/anvilogic-as

# Explore the embedded API-surface registry
anvilogic-as registry list --domain eoi
anvilogic-as registry describe detections.deploy
anvilogic-as registry validate

# Configure a credential (platform UI: Settings > Generate API Key)
anvilogic-as auth set --api-key <KEY>   # OS keychain; config-file fallback (0600)
anvilogic-as auth status
anvilogic-as auth validate

# Generic API driver: raw path or registry endpoint id.
# NOTE: the /v1/... paths below are mock/placeholder examples - no real
# Anvilogic path is publicly known (api-surface/ records path UNKNOWN
# everywhere), and they only resolve against testdata/fixtures under
# ANVILOGIC_MOCK_MODE=1.
anvilogic-as api GET detections.list             # resolves via the registry (exits 1 until Wave 3 captures paths)

# Offline mock mode (fixtures in testdata/fixtures/)
ANVILOGIC_MOCK_MODE=1 anvilogic-as api GET /v1/ping
ANVILOGIC_MOCK_MODE=1 anvilogic-as api GET /v1/items --paginate --paginate-style page_number
```

## The api-surface/ knowledge base

`api-surface/` is the Wave-1 harvest of everything publicly known about the
Anvilogic SaaS API: `endpoints.yaml` (the machine-readable registry),
`auth.md`, `conventions.md`, `domain-model.md`, `coverage.yaml`, and
`sources.md`. **No public source shows an actual Anvilogic REST path or
method**, so every Wave-1 entry records `method: UNKNOWN` / `path: UNKNOWN`
with per-entry `confidence` and `evidence[]` citations — nothing is
invented. The CLI treats the registry as data: calling an endpoint id whose
path is unknown exits `1` with a hint to pass an explicit path until Wave 3
captures the real ones. Override the embedded copy with `--registry <file>`
or `ANVILOGIC_REGISTRY_PATH`.

Because the Authorization scheme is also unconfirmed publicly, the header
scheme is configurable via the `auth_scheme` config key (default `Bearer`);
see `api-surface/auth.md`.

### Configuration cascade (first hit wins)

`ANVILOGIC_API_KEY` / `ANVILOGIC_BASE_URL` / `ANVILOGIC_PROFILE` env vars →
OS keychain (service `anvilogic-as`, account = profile) →
`.claude/settings.local.json` → `.claude/settings.json` (key `anvilogic`) →
`~/.config/anvilogic-as/config.json` (0600) → defaults.

This follows the family convention (env → keychain → settings.local.json →
settings.json → defaults, as in `jira-as`) with one documented extension:
the `~/.config/anvilogic-as/config.json` layer just before defaults,
analogous to jira-as's JSON credential-file fallback.

## Wave status

| Wave | Deliverable | Status |
|------|-------------|--------|
| 1 | Public-source API-surface harvest (`api-surface/`) | done |
| 2 | v0 CLI scaffold (this repo): registry-driven `api` driver, auth, config, mock mode | done |
| 3 | Capture real endpoint paths/methods; fill in `path`/`pagination`/`fixture` | pending |
| 4 | Live validation against the platform; recorded fixtures; `auth.probe` | pending |
| 5 | Distribution: Homebrew tap (`grandcamel/homebrew-tap`), release polish | pending |

## Development

```sh
make build   # bin/anvilogic-as with version injected
make test    # go test -race ./...
make lint    # golangci-lint run
make smoke   # mock-mode ping
```

Releases are automated with release-please + GoReleaser
(darwin/linux/windows x amd64/arm64).

## License

MIT — see [LICENSE](LICENSE).
