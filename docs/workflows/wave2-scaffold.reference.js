export const meta = {
  name: 'anvilogic-wave2-scaffold',
  description: 'Scaffold anvilogic-as Go CLI and Anvilogic-Assistant-Skills plugin skeleton',
  phases: [
    { title: 'Scaffold', detail: 'Go repo + plugin repo in parallel' },
    { title: 'Verify', detail: 'build/lint/mock smoke + conventions parity' },
    { title: 'Fix', detail: 'bounded fix loop (max 2 rounds)' },
  ],
}

// Constants baked in deliberately — the Wave-1 run showed `args` can arrive as a string.
const DATE = '2026-07-06'
const GO_REPO = '/Users/jasonkrueger/projects/anvilogic-as'
const PLUGIN_REPO = '/Users/jasonkrueger/projects/Anvilogic-Assistant-Skills'

const SCAFFOLD_SCHEMA = {
  type: 'object',
  required: ['files_created', 'build_status', 'test_status', 'notes'],
  properties: {
    files_created: { type: 'number' },
    build_status: { type: 'string', description: 'pass|fail|n/a with detail' },
    test_status: { type: 'string', description: 'pass|fail|n/a with detail' },
    notes: { type: 'string' },
  },
}

const ISSUES_SCHEMA = {
  type: 'object',
  required: ['pass', 'issues'],
  properties: {
    pass: { type: 'boolean' },
    issues: {
      type: 'array',
      items: {
        type: 'object',
        required: ['repo', 'file', 'description', 'severity'],
        properties: {
          repo: { type: 'string', enum: ['anvilogic-as', 'Anvilogic-Assistant-Skills'] },
          file: { type: 'string' },
          description: { type: 'string' },
          severity: { type: 'string', enum: ['blocker', 'major', 'minor'] },
        },
      },
    },
  },
}

const FIX_SCHEMA = {
  type: 'object',
  required: ['fixed', 'notes'],
  properties: { fixed: { type: 'number' }, notes: { type: 'string' } },
}

const GO_PROMPT = `Scaffold the complete v0 Go CLI at ${GO_REPO} (directory exists; api-surface/ already contains Wave-1 artifacts incl. endpoints.yaml — do NOT modify api-surface/ content). Go 1.26 is installed. Do NOT git init or push; a later step handles git.

Module: github.com/grandcamel/anvilogic-as. CLI framework: Cobra (spf13/cobra). Binary name: anvilogic-as.

## Architecture (follow exactly)

Layout:
- cmd/anvilogic-as/main.go — thin main; version injected via -ldflags -X main.version (default "dev", fall back to debug.ReadBuildInfo()).
- embed.go at MODULE ROOT (package anvilogicas) with //go:embed api-surface/endpoints.yaml exposing the embedded bytes. (go:embed cannot reference parent dirs, so the embed directive must live at the root package, which internal/registry imports.)
- internal/cli/ — cobra wiring: root.go (global flags --output json|ndjson|text default json, --profile, --timeout, --registry, --base-url), api.go, auth.go, config.go, registry.go, version.go.
- internal/config/ — cascade resolution (first hit wins): env ANVILOGIC_API_KEY / ANVILOGIC_BASE_URL / ANVILOGIC_PROFILE > OS keychain (zalando/go-keyring, service "anvilogic-as", account = profile name) > .claude/settings.local.json > .claude/settings.json (JSON key "anvilogic": {"base_url":..., "profile":...}) > ~/.config/anvilogic-as/config.json (created 0600) > defaults. Profiles = named sections {base_url, auth_scheme}.
- internal/auth/ — TokenSource interface: Token(ctx context.Context) (string, error). Implementations: static (from config cascade). The Authorization header SCHEME must be configurable (config key auth_scheme, default "Bearer") because the Bearer form is NOT yet confirmed publicly — see api-surface/auth.md.
- internal/httpx/ — client wrapping net/http: 30s default timeout, context cancellation (SIGINT → cancel), User-Agent "anvilogic-as/<version> (<os>/<arch>)", exponential backoff + jitter (base 500ms, max 4 attempts) on 429/5xx/transient net errors honoring Retry-After; non-idempotent methods (POST/PATCH) retry ONLY on 429-with-Retry-After.
- internal/paginate/ — Paginator interface { Next(prevReq *http.Request, respBody []byte) (nextReq *http.Request, items []json.RawMessage, done bool, err error) } (adjust signature sensibly) with implementations: none, offsetLimit, pageNumber, cursor, linkHeader. Style selected from registry entry pagination.style; explicit --paginate on a raw path REQUIRES --paginate-style (else exit 1 with hint).
- internal/registry/ — types matching api-surface/endpoints.yaml schema (version, base_url, generated, endpoints map: domain, method, path, summary, params, pagination.style, auth, risk, confidence, evidence[], first_seen, last_verified — parse leniently, unknown fields ignored; note fixture key field OPTIONAL since Wave-1 entries lack it). Loader: --registry flag / ANVILOGIC_REGISTRY_PATH env overrides the embedded copy. registry validate command checks schema + enum values.
- internal/output/ — writers: json (pretty to stdout), ndjson (one object/line), text (minimal human summary). stdout = DATA ONLY. stderr = diagnostics + structured error JSON {"error":{"code":"...","status":N,"hint":"..."}}. Exit codes: 0 ok, 1 validation, 2 auth, 3 permission, 4 not-found, 5 rate-limit-exhausted, 6 conflict, 7 server.
- internal/mock/ — ANVILOGIC_MOCK_MODE=1 swaps an http.RoundTripper: match request method+path first against registry entries with a concrete path, else against testdata/fixtures/<METHOD>_<path-with-slashes-as-underscores>.json (e.g. GET /v1/ping → GET_v1_ping.json). Unmatched → 404-style response so the CLI exits 4 with hint "add fixture or registry entry". Ship fixtures: GET_v1_ping.json ({"status":"ok","message":"anvilogic-as mock"}) and a two-page cursor pair for pagination unit tests.

## Command surface
- api <METHOD> <path-or-registry-id> [--query k=v]... [--data @file|inline-json] [--paginate] [--paginate-style s] — the generic driver. Registry-id shorthand: if the arg matches an endpoint id, resolve its path; entries with path UNKNOWN exit 1 with hint "endpoint path not yet captured (Wave 3); pass an explicit path". Injects auth header via TokenSource; agent never sees token mechanics.
- auth set (prompt or --api-key; stores via keyring, fallback config file 0600) | auth status (which source provided the key, masked) | auth validate (probes registry auth.probe endpoint if set; with no key configured exits 2 with hint including where to get a key: platform Settings > Generate API Key).
- config get|set|list [--profile].
- registry list [--domain], registry describe <id> (full entry incl. evidence + confidence), registry validate.
- version [--check-min X.Y.Z] (exit 1 if below).

## Also create
- go.mod (go 1.26), README.md (project intent: minimalist agent-first CLI, registry-driven; quick start; the api-surface/ knowledge-base explanation; wave status table), LICENSE (MIT, holder Jason Krueger ${DATE.slice(0, 4)}), .gitignore (api-surface/.private/, bin/, dist/, coverage.out), Makefile (build/test/lint/smoke targets), .golangci.yml (modest: govet, staticcheck, errcheck, gofmt), .goreleaser.yaml (darwin/linux/windows × amd64/arm64, ldflags version inject, archives, checksums; brews section for tap grandcamel/homebrew-tap commented out until Wave 5), release-please-config.json + .release-please-manifest.json (release-type go, start 0.1.0), .github/workflows/ci.yml (push/PR: go build, go vet, golangci-lint-action, go test -race ./..., and the mock smoke: ANVILOGIC_MOCK_MODE=1 go run ./cmd/anvilogic-as api GET /v1/ping), .github/workflows/release.yml (release-please + goreleaser on tag).
- Unit tests: config cascade precedence; httpx retry/backoff incl. Retry-After honored + POST-not-retried-on-500 (httptest); every paginator style walks the two-page fixtures; registry load/validate of the REAL embedded api-surface/endpoints.yaml; output exit-code mapping; mock transport matching. Golden files where useful.

## Acceptance (run these yourself before returning; all must pass)
1. go build ./... && go vet ./... && go test ./...
2. ANVILOGIC_MOCK_MODE=1 go run ./cmd/anvilogic-as api GET /v1/ping → fixture JSON on stdout, exit 0.
3. ANVILOGIC_MOCK_MODE=1 go run ./cmd/anvilogic-as api GET detections.list → exit 1, hint mentions Wave-3/unknown path.
4. go run ./cmd/anvilogic-as registry list --domain eoi → the 8 eoi ids; registry describe detections.deploy shows evidence; registry validate passes on embedded registry.
5. env -u ANVILOGIC_API_KEY go run ./cmd/anvilogic-as auth validate → exit 2, hint on stderr, stdout empty.
Keep the code minimal and idiomatic — no speculative abstractions beyond the seams named above. Return the structured summary.`

const PLUGIN_PROMPT = `Scaffold the doc-only Claude Code plugin repo at ${PLUGIN_REPO} (create the directory). This is the skills/plugin sibling of the Go CLI repo ${GO_REPO} (binary: anvilogic-as). Do NOT git init or push. Wave 2 = SKELETON ONLY: structure + stubs; real domain skills are Wave 4.

Mirror the conventions of grandcamel/JIRA-Assistant-Skills (doc-only plugin: skills contain no code; the CLI does the work). Fetch reference files from GitHub as needed (gh api repos/grandcamel/JIRA-Assistant-Skills/... with raw accept header) — especially .claude-plugin/plugin.json, a representative skills/*/SKILL.md, and commands/jira-assistant-setup.md.

Create:
- .claude-plugin/plugin.json — name "anvilogic-assistant-skills", version 0.1.0, description (Anvilogic SOC detection-engineering automation via the anvilogic-as CLI), author, keywords.
- .claude-plugin/marketplace.json — single plugin, source "./", category matching the JIRA repo's convention.
- skills/anvilogic-assistant/SKILL.md — router-hub STUB: frontmatter (name, description with trigger phrases, third person, <1024 chars; allowed-tools ["Bash","Read","Glob","Grep"]), body explaining the registry-driven approach: run 'anvilogic-as registry list'/'registry describe <id>' for discovery, 'anvilogic-as api ...' as generic driver, risk marks legend (- read / ! write / !! bulk / !!! destructive rendered as the warning-sign emoji marks used in JIRA-Assistant-Skills), a Quick Reference table of the 9 domains (detections, scenarios, eoi, hunting, alerts, feeds, forge, blueprints, metrics) each marked "Wave 4 — coming", and a WAVE STATUS callout that endpoint paths are not yet captured (mock mode + registry exploration only today).
- commands/anvilogic-assistant-setup.md — setup slash command: check binary (command -v anvilogic-as; anvilogic-as version --check-min 0.1.0), install instructions (go install github.com/grandcamel/anvilogic-as/cmd/anvilogic-as@latest now; brew tap grandcamel/tap noted as coming at first release), credential setup (ANVILOGIC_API_KEY env or 'anvilogic-as auth set'; key from platform Settings > Generate API Key, admin role), validation (anvilogic-as auth validate; ANVILOGIC_MOCK_MODE=1 fallback demo), troubleshooting table keyed to exit codes 1-7.
- README.md — what this is, relationship to anvilogic-as repo and api-surface registry, wave roadmap table (Wave 1 done ${DATE}, Wave 2 scaffold, 3 gated capture, 4 coverage, 5 release), install/quick-start.
- LICENSE (MIT, Jason Krueger), .gitignore (.claude/settings.local.json, .private/), release-please-config.json + manifest (release-type simple, 0.1.0), .github/workflows/ci.yml (validate plugin.json + marketplace.json with jq, markdown lint via npx markdownlint-cli2 with a permissive .markdownlint.yaml, check SKILL.md frontmatter has name+description).
Acceptance: jq empty on both JSON manifests; SKILL.md frontmatter parses (yq); every file references the anvilogic-as binary consistently; no invented API endpoints/paths anywhere (registry is the only source of truth). Return the structured summary.`

phase('Scaffold')
log('Scaffolding Go CLI and plugin repos in parallel')
const [goRes, pluginRes] = await parallel([
  () => agent(GO_PROMPT, { label: 'scaffold:anvilogic-as', phase: 'Scaffold', schema: SCAFFOLD_SCHEMA, effort: 'high' }),
  () => agent(PLUGIN_PROMPT, { label: 'scaffold:plugin', phase: 'Scaffold', schema: SCAFFOLD_SCHEMA }),
])
if (!goRes) throw new Error('Go scaffold agent failed')
log(`Go repo: ${goRes.files_created} files, build ${goRes.build_status}, tests ${goRes.test_status}; plugin: ${pluginRes ? pluginRes.files_created + ' files' : 'FAILED'}`)

const VERIFIER_PROMPT = `Integration-verify the freshly scaffolded repos. Run commands, do NOT edit files.

Repo 1: ${GO_REPO} (Go CLI). From that directory run and CHECK:
1. go build ./... ; go vet ./... ; go test -race ./...
2. gofmt -l . (excluding vendor) must be empty; if golangci-lint is on PATH, golangci-lint run (report findings as issues; tool missing = note, not issue).
3. Mock smoke: ANVILOGIC_MOCK_MODE=1 go run ./cmd/anvilogic-as api GET /v1/ping (expect fixture JSON stdout, exit 0, nothing but data on stdout — pipe stderr separately to check).
4. ANVILOGIC_MOCK_MODE=1 go run ./cmd/anvilogic-as api GET detections.list → exit 1 with a Wave-3/unknown-path hint on stderr.
5. go run ./cmd/anvilogic-as registry list --domain eoi (expect 8 ids), registry describe detections.deploy (shows confidence inferred + evidence), registry validate (pass).
6. env -u ANVILOGIC_API_KEY go run ./cmd/anvilogic-as auth validate → exit 2, structured error JSON on stderr, empty stdout.
7. Confirm api-surface/ Wave-1 files are byte-identical to git-less originals in spirit: they must NOT have been modified (check no new keys/rewrites: spot-check endpoints.yaml still has 33 endpoints via yq '.endpoints | length' and auth.md unchanged header).
8. .github/workflows/ci.yml steps match what you just ran (no phantom steps that would fail in CI, e.g. referencing missing Make targets); .goreleaser.yaml parses (goreleaser not installed — yq/yaml parse only); release-please config valid JSON.

Repo 2: ${PLUGIN_REPO} (doc-only plugin):
9. jq empty .claude-plugin/plugin.json .claude-plugin/marketplace.json; SKILL.md frontmatter has name + description (<1024 chars); no code files (skills are docs only).
10. Cross-repo consistency: plugin references binary name anvilogic-as, install paths github.com/grandcamel/anvilogic-as, version floor 0.1.0 consistent with release-please manifests.
Severity: anything that would break CI or the smoke contract = blocker; contract drift (wrong exit code, stdout pollution) = blocker; style/docs = minor. Return pass + issues.`

const CRITIC_PROMPT = `Conventions-parity critique of the two scaffolded repos vs the user's established Assistant-Skills family (compare against grandcamel/JIRA-Assistant-Skills and grandcamel/Assistant-Skills conventions on GitHub — fetch reference files with gh api as needed). Do NOT edit files.

Check ${PLUGIN_REPO}: plugin.json/marketplace.json field parity with the JIRA plugin (naming style anvilogic-assistant-skills, category, structure); SKILL.md follows conventions (frontmatter name/description with trigger phrases, no deprecated when_to_use field, risk-mark legend present, <500 lines); setup command structure parity with jira-assistant-setup.md; README wave-status honesty (must NOT claim working endpoint coverage — paths are all UNKNOWN until Wave 3).
Check ${GO_REPO}: README honesty (0 confirmed endpoints, mock-mode-only today, must not oversell); .gitignore contains api-surface/.private/; exit-code taxonomy documented and IDENTICAL to jira-as (1 validation / 2 auth / 3 permission / 4 not-found / 5 rate-limit / 6 conflict / 7 server); config cascade documented and matches the family convention (env > keychain > settings.local.json > settings.json > defaults); no gated/customer-only Anvilogic content anywhere (only public facts already in api-surface/).
Severity: leaked gated content or overclaiming = blocker; convention drift = major; polish = minor. Return pass + issues.`

let clean = false
let remaining = []
for (let round = 0; round < 2; round++) {
  phase('Verify')
  const [iv, cc] = await parallel([
    () => agent(VERIFIER_PROMPT, { label: `verify:integration-r${round + 1}`, phase: 'Verify', schema: ISSUES_SCHEMA }),
    () => agent(CRITIC_PROMPT, { label: `verify:conventions-r${round + 1}`, phase: 'Verify', schema: ISSUES_SCHEMA }),
  ])
  const issues = [...(iv ? iv.issues : []), ...(cc ? cc.issues : [])]
  const actionable = issues.filter((i) => i.severity !== 'minor')
  remaining = issues
  if (actionable.length === 0) {
    clean = true
    log(`Verification clean on round ${round + 1} (${issues.length} minor notes)`)
    break
  }
  log(`Round ${round + 1}: ${actionable.length} actionable issues — fixing`)
  await agent(
    `Fix these verified issues in the scaffolded repos (${GO_REPO} and ${PLUGIN_REPO}). After fixing, re-run the acceptance commands relevant to what you changed (at minimum: go build ./... && go test ./... in the Go repo if it was touched). Do not modify api-surface/ Wave-1 artifacts. Issues (JSON): ${JSON.stringify(actionable)}
Minor notes for context: ${JSON.stringify(issues.filter((i) => i.severity === 'minor'))}
Return counts.`,
    { label: `fix:round${round + 1}`, phase: 'Fix', schema: FIX_SCHEMA, effort: 'high' }
  )
}

return {
  go: goRes,
  plugin: pluginRes,
  verificationClean: clean,
  remainingIssues: remaining,
}