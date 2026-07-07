# Wave 5 Runbook — Live Validation + First Release

Validates the CLI against a real tenant, records sanitized fixtures, and
ships v0.1.0 of both repos.

**Entry:** Wave 4 complete; an Anvilogic API key for a tenant you are
authorized to test against (platform Settings → Generate API Key, admin
role). A docs login is NOT sufficient — this wave needs a real key.
**Exit:** live read-only smoke green; fixtures refreshed from sanitized real
responses; coverage rows → `tested: true`; `last_verified` stamped;
`anvilogic-as` v0.1.0 released (goreleaser artifacts + Homebrew tap);
plugin listed.

## Safety rails

- API key only via env (`ANVILOGIC_API_KEY`) or `anvilogic-as auth set`
  (keychain). Never in files, never in fixtures, never in CI logs.
- **Read-only first, mutating never by default.** Mutating/destructive
  endpoints are exercised only with explicit human approval per risk mark,
  against non-production content, with cleanup steps identified BEFORE the
  call.
- Every recorded fixture passes a sanitization check before commit: no
  tenant ids/names, hostnames beyond the public `*.anvilogic.com`, emails,
  usernames, API keys, or detection content that could identify the tenant.
  Replace with synthetic equivalents preserving shape.

## Steps (orchestrated)

1. **Auth confirmation:** `anvilogic-as auth validate` against the live
   tenant. If the header scheme differs from the configured `auth_scheme`,
   fix the default in `internal/config/` — this is the last chance to catch
   it before release.
2. **Read-only smoke fan-out:** for every `confidence: confirmed`, `risk:
   read` registry entry, run `anvilogic-as api GET <id> --paginate` live.
   Record: HTTP status, response-envelope shape vs fixture, pagination
   behavior vs configured style, rate-limit headers observed (fill the
   `conventions.md` rate-limit section with CONFIRMED values). Failures →
   fix registry/config → re-run (max 2 rounds). Flip `tested: true` and
   stamp `last_verified` only for endpoints that passed.
3. **Fixture refresh:** regenerate `testdata/fixtures/` from sanitized live
   shapes; a verifier agent scans the diff for tenant identifiers before
   commit; `go test ./...` still green.
4. **Mutating-op validation (optional, gated):** only ops the human approves,
   one at a time, cleanup verified.
5. **Release `anvilogic-as`:** merge the release-please PR (v0.1.0) → tag
   push triggers goreleaser → verify GitHub Release artifacts
   (darwin/linux/windows × amd64/arm64, checksums) → create
   `grandcamel/homebrew-tap` repo, un-comment the `brews:` section in
   `.goreleaser.yaml`, re-release or cut 0.1.1 so the tap formula publishes →
   `brew install grandcamel/tap/anvilogic-as` smoke on a clean shell →
   `go install github.com/grandcamel/anvilogic-as/cmd/anvilogic-as@latest`
   smoke.
6. **Release the plugin:** update setup command install order (brew first,
   go install fallback), merge its release-please PR, verify the marketplace
   manifests, add the plugin to the personal marketplace listing pattern used
   by the other Assistant-Skills families.
7. **Close the loop:** README wave tables in both repos updated to "complete";
   `coverage.yaml` is now the living coverage dashboard for future API
   additions.

## Verification gate

Fresh terminal, no repo checkout: install via brew, `anvilogic-as version`,
`anvilogic-as auth set` + `auth validate` (exit 0), one live read
(`api GET <cheap-confirmed-id>`), and `ANVILOGIC_MOCK_MODE=1` demo path all
work. That is the end-user acceptance test.
