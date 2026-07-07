# Wave 4 Runbook — Coverage Implementation

Turns the confirmed registry into working coverage: domain skills in the
plugin repo, fixtures and per-endpoint config in this repo, and ergonomic CLI
verbs **only where genuinely valuable**. Runs as an orchestrated workflow.

**Entry:** Wave 3 merged — registry majority `confirmed`, `conventions.md`
sections CONFIRMED, `registry validate` passes, CI green.
**Exit:** every `captured: true` row in `api-surface/coverage.yaml` is also
`implemented: true` and `documented: true`; plugin skills complete; both CIs
green.

## Guardrail: don't recreate the jira-as trap

The whole design premise is that the agent drives `anvilogic-as api` +
`registry describe` directly. jira-as accumulated ~209 hand-written
per-endpoint command sites; that is the failure mode to avoid.

- **Default deliverable per endpoint:** a registry entry with a confirmed
  path/pagination config, a mock fixture (`fixture:` key +
  `testdata/fixtures/<name>.json`), and skill documentation showing the
  `api` invocation. NO new Go code.
- **Ergonomic CLI verbs** (e.g. `anvilogic-as detections list`) are allowed
  only when they add real value over the generic driver (multi-call
  composition, heavy client-side shaping, high-frequency + destructive ops
  needing extra guards). Each one needs a one-line justification in the PR.
  If more than ~5 emerge, switch to registry-driven codegen instead of
  hand-writing them.

## Work breakdown (orchestrated)

1. **Gap query** (plain code, not an agent): parse `coverage.yaml`, list rows
   with `captured: true, implemented: false`, grouped by domain
   (detections, scenarios, eoi, hunting, alerts, feeds, forge, blueprints,
   metrics).
2. **Fan-out per-domain implementer pairs** — both members of a pair read the
   SAME registry rows:
   - *CLI-side agent* (this repo): fixtures for each confirmed endpoint
     (realistic but fully synthetic data — never captured tenant data),
     pagination config verified against `conventions.md`, `auth.probe`
     endpoint set once a cheap confirmed GET exists, paginator tests extended
     to the confirmed style(s).
   - *Skill agent* (plugin repo): `skills/anvilogic-<domain>/SKILL.md` —
     frontmatter (name, third-person description with trigger phrases,
     <1024 chars; `allowed-tools: ["Bash","Read","Glob","Grep"]`), body <500
     lines, L3 detail in `docs/` per progressive disclosure, risk marks from
     the registry `risk` enum (read `-`, write ⚠️, bulk ⚠️⚠️, destructive
     ⚠️⚠️⚠️), every documented invocation copy-paste runnable in mock mode.
3. **Synthesis** (after all domains land): rewrite the router-hub
   `skills/anvilogic-assistant/SKILL.md` quick-reference table (domain →
   skill → risk profile), finalize `commands/anvilogic-assistant-setup.md`,
   bump plugin.json/marketplace.json wording (drop "Wave 4 — coming").
4. **Adversarial review fan-out:**
   - *Drift check*: every skill invocation actually runs (mock mode) with the
     documented flags and exit codes; `registry describe` ids referenced in
     skills all exist.
   - *Risk audit*: every destructive/bulk op carries the right mark and a
     confirmation-pattern note; no skill soft-pedals a `destructive` row.
   - *Structure lint*: SKILL.md <500 lines, frontmatter valid, no deprecated
     `when_to_use`, progressive disclosure respected.
   - *Registry integrity*: `registry validate`; coverage flags only flipped
     for rows with real fixtures/docs; upgrade rule respected (no deleted
     ids).
5. **Bounded fix loop** (max 2 rounds), then flip `implemented`/`documented`
   flags in `coverage.yaml` and commit both repos (conventional commits;
   release-please picks them up — still no release tag until Wave 5).

## Verification gate

```sh
make build && make test && make lint
./bin/anvilogic-as registry validate
ANVILOGIC_MOCK_MODE=1 ./bin/anvilogic-as api GET <every-newly-fixtured-id>  # scripted loop
```
Plugin repo CI (manifest jq checks, frontmatter check, markdownlint) green;
run the plugin-validator agent if available. Degraded note: if any domain is
still unconfirmed post-Wave 3, implement it mock-only, mark its skill
"unverified endpoint" ⚠️, and leave `implemented: false`.
