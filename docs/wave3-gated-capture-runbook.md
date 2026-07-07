# Wave 3 Runbook — Gated API-Reference Capture

Wave 3 upgrades the endpoint registry from `inferred` to `confirmed` using the
customer-gated Anvilogic REST API reference. It must run on a machine with an
authenticated docs session (customer SSO). This runbook is self-contained on
purpose: the machine running it may not share local state with the machine
that ran Waves 1–2 — the repos on GitHub are the source of truth.

## Hard rules (read first)

1. **Raw gated captures never leave `api-surface/.private/`** (gitignored).
   The public registry gets *distilled facts only* — method, path, params,
   pagination fields — never verbatim gated prose, examples, or screenshots.
2. **Registry upgrades are id-matched merges.** Never delete an endpoint id.
   Raise `confidence`, fill `UNKNOWN` fields, append `evidence` entries
   (`kind: gated-docs`, `source:` pointing at a `.private/` capture file).
   Deprecate dead entries with `status: retired`.
3. Your docs access runs through an employer customer license — capture only
   what you need for engineering reference, and keep it private per rule 1.

## Prerequisites (work machine)

- Clone both repos:
  `gh repo clone grandcamel/anvilogic-as && gh repo clone grandcamel/Anvilogic-Assistant-Skills`
- Go 1.26+ (`make build && make test` should pass), Claude Code with a
  browser-automation tool (claude-in-chrome or Playwright) attached to a
  Chrome profile that is **already logged in** to `docs.anvilogic.com`
  (login flows through `secure.anvilogic.com`; do not automate the SSO login
  itself).
- Confirm the gate is open: browsing to
  `https://docs.anvilogic.com/rest-api/api-reference` shows the reference,
  not a login redirect.

## Step 1 — Probe for a backing OpenAPI spec (cheapest win)

The gated docs are GitBook-powered; GitBook "API reference" sections are
usually rendered from an OpenAPI file. In the authenticated browser session:

1. Open the api-reference page with DevTools → Network. Look for fetches of
   `*.json`/`*.yaml` containing `openapi`, `swagger`, or `spec` (GitBook often
   serves these from its CDN once the page renders).
2. Also try, while authenticated: `docs.anvilogic.com/llms.txt`,
   `llms-full.txt`, `sitemap.md` (the public site exposes these; the gated
   site may too).
3. If a spec is found: save it to `api-surface/.private/openapi/` and skip to
   Step 3 — a spec supersedes page scraping entirely.

## Step 2 — Page-by-page capture (fallback)

Walk the api-reference navigation tree. For each endpoint page, record into
`api-surface/.private/captures/<domain>/<page-slug>.md`: method, path, params
(name/in/type/required), request/response field names, pagination mechanics,
error envelope, rate-limit headers. Prefer the page's structured blocks
(`get_page_text`) over screenshots. Also capture the auth page — the exact
`Authorization` header scheme is still unconfirmed (see `api-surface/auth.md`);
if it is not `Bearer`, set the `auth_scheme` config default accordingly in
`internal/config/`.

High-value targets recorded during Wave 1 (see `api-surface/sources.md`):
deploy-a-detection-pack, import-existing-rules, snowflake-data-ingestion,
forward-events, manage-users, authentication-settings.

## Step 3 — Registry upgrade

Run as an orchestrated workflow (or manually for small deltas):

1. **Upgrade agent**: merge captured facts into `api-surface/endpoints.yaml`
   per the hard rules above. Fill `conventions.md` sections (pagination,
   rate limits, error envelope, versioning) and flip their banners
   `UNKNOWN → CONFIRMED`. Update `auth.md`. Set `last_verified` dates. Add
   new endpoints the capture revealed (new ids, `confidence: confirmed`).
   Update `coverage.yaml`: captured rows get `registry_id` + `captured: true`.
2. **Adversarial verifier**: every upgraded field must cite a `.private/`
   capture file that actually contains it; every `confirmed` entry needs
   `kind: gated-docs` or `live-response` evidence; scan the *public* diff for
   leaked gated prose (`git diff -- api-surface/ ':!api-surface/.private'`).
3. **Delta report**: compare Wave-1 inferences vs confirmed reality (which
   Tines-derived guesses were right?) — write to `api-surface/.private/`.

## Step 4 — Validate against the binary

```sh
make build
./bin/anvilogic-as registry validate            # schema still parses
./bin/anvilogic-as registry list --domain detections
ANVILOGIC_MOCK_MODE=1 ./bin/anvilogic-as api GET <some-now-confirmed-id>
```

If pagination styles were confirmed, add real fixture shapes under
`testdata/fixtures/` and extend the paginator tests to match.

## Step 5 — Ship

Commit only distilled artifacts (`api-surface/*.yaml`, `*.md` — verify
`.private/` is absent from `git status`), push, confirm CI green. Wave 4
(coverage implementation + domain skills) is unblocked once the registry is
majority-confirmed.
