# Anvilogic Assistant Skills — Project Plan

> **Status (2026-07-07):** Waves 1–2 are **complete** (this repo + the
> [Anvilogic-Assistant-Skills](https://github.com/grandcamel/Anvilogic-Assistant-Skills)
> plugin skeleton, CI green). Wave 3 gated-docs capture has been **run via
> Playwright with an authenticated customer SSO session** on an authorized
> machine; the registry upgrade and all remaining waves continue in that
> session. Per-wave runbooks live next to this file:
> [wave3](wave3-gated-capture-runbook.md) ·
> [wave4](wave4-coverage-runbook.md) ·
> [wave5](wave5-release-runbook.md).
> Reference workflow scripts from Waves 1–2 are under [workflows/](workflows/).

## Context

This is an "Assistant Skills" family (like JIRA, Confluence, Splunk before
it): a doc-only Claude Code plugin repo of SKILL.md skills plus a real CLI
that skills invoke via Bash. This family targets **Anvilogic** (SOC
detection-engineering SaaS) and deliberately improves on the Python design:

- **Go instead of Python** — a single static binary kills the known jira-as
  pain points: pip/venv fragility, ~209 hand-written client-instantiation
  sites, mock/real signature drift, template path-guessing.
- **Minimalist wrapper, not command-per-endpoint** — the CLI solves the
  *idiomatic* API problems once (auth injection, token lifecycle masking,
  pagination, retry/backoff, rate limits, JSON output, exit codes) and
  exposes a generic `api` driver. The agent's innate REST understanding does
  the rest, guided by a machine-readable **endpoint registry** artifact.
- **Artifact-first waves** — Wave 1 captured API-surface artifacts (endpoint
  definitions, conventions, coverage matrix) that later waves consume as the
  coverage goal.

### Locked decisions
- Language: **Go** (Cobra). Two public repos: `anvilogic-as` (CLI + the
  `api-surface/` knowledge base) and `Anvilogic-Assistant-Skills` (doc-only
  plugin).
- Coverage: everything reachable; priority detections/content → deploy &
  repos → alerts & hunting.
- Gated/customer-only content **never** enters the public repos: raw captures
  live in gitignored `api-surface/.private/`; the public registry carries
  distilled facts only (method/path/params).

### Research findings (Wave 0, verified 2026-07-06)
- **Auth**: static API key, generated in UI (Settings → Generate API Key,
  admin role). Host `secure.anvilogic.com`. The exact `Authorization` header
  scheme was **not publicly confirmed** — the CLI's `auth_scheme` config key
  (default `Bearer`) exists for this reason; Wave 3 confirms or corrects it.
- **API reference is gated**: `docs.anvilogic.com/rest-api/api-reference`
  (GitBook, customer login). No public OpenAPI spec.
- **Public sources used by Wave 1**: `public-docs.anvilogic.com`
  (`llms.txt`/`llms-full.txt`/`sitemap.md`), `github.com/anvilogic-forge/armory`,
  the Azure Sentinel Anvilogic solution (ADX-side only — excluded from the
  SaaS registry), SOAR connector pages (Tines was the richest signal).
- **Domain nouns**: detections (threat identifiers), threat scenarios, events
  of interest (EOI), hunting, alerts, data feeds/repositories
  (Splunk/Snowflake/Databricks/Azure), Forge/Armory, blueprints,
  MTTD/maturity metrics.

## Waves

### Wave 1 — Public artifact capture ✅ (2026-07-06)
Delivered `api-surface/`: `endpoints.yaml` (33 evidence-cited entries — 0
confirmed / 32 inferred / 1 unknown; every method/path UNKNOWN because no
public source shows real SaaS REST paths), `coverage.yaml` (64 rows),
`domain-model.md`, `auth.md`, `conventions.md` (all convention sections
honestly UNKNOWN), `sources.md` (provenance + Wave-3 target list).
**Registry upgrade rule** (binding for all later waves): only raise
`confidence`, fill `UNKNOWN` fields, and append `evidence`; never delete ids
(deprecate with `status: retired`).
Workflow shape: 4 parallel harvesters → synthesis → adversarial evidence
verifier + completeness critic → bounded fix loop (max 2 rounds). See
`workflows/wave1-capture.reference.js`.

### Wave 2 — Scaffold + Go CLI core ✅ (2026-07-06)
Both repos public with CI green. CLI core: generic `api` driver,
config cascade (env > keychain > `.claude/settings.local.json` >
`.claude/settings.json` > `~/.config/anvilogic-as/config.json` > defaults),
`TokenSource` auth seam, five paginator styles selected per-endpoint from the
registry, registry embedded via root `embed.go` (`--registry` /
`ANVILOGIC_REGISTRY_PATH` override), mock transport
(`ANVILOGIC_MOCK_MODE=1`) matching at the RoundTripper layer so mock parity
drift is structurally impossible. Exit codes (family-wide contract): 0 ok /
1 validation / 2 auth / 3 permission / 4 not-found / 5 rate-limit /
6 conflict / 7 server. stdout = data only; stderr = diagnostics + structured
error JSON. See `workflows/wave2-scaffold.reference.js`.

### Wave 3 — Gated capture 🔄 (in progress on authorized machine)
Runbook: [wave3-gated-capture-runbook.md](wave3-gated-capture-runbook.md).
Crawl done via Playwright + customer SSO. Remaining: registry upgrade
(id-matched merges only), conventions/auth banners UNKNOWN → CONFIRMED,
adversarial verification (every upgrade cites a `.private/` capture; public
diff scanned for leaked gated prose), delta report vs Wave-1 inferences,
`registry validate` + binary smoke.

### Wave 4 — Coverage implementation
Runbook: [wave4-coverage-runbook.md](wave4-coverage-runbook.md).
Registry-driven skills + fixtures + (only-where-valuable) ergonomic CLI
verbs; every `captured` coverage row → `implemented` + `documented`.

### Wave 5 — Live validation + release
Runbook: [wave5-release-runbook.md](wave5-release-runbook.md).
Live read-only smoke against a tenant API key, sanitized fixtures,
coverage rows → `tested`, first release (release-please → goreleaser →
Homebrew tap), marketplace listing.

**Cross-cutting risks:** SOAR action names may not map 1:1 to REST paths
(stay `inferred` until confirmed); ADX/Sentinel surface must never enter the
SaaS registry; the public repos must never leak gated content.
