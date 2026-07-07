export const meta = {
  name: 'anvilogic-wave1-capture',
  description: 'Capture public Anvilogic API-surface artifacts into api-surface/',
  phases: [
    { title: 'Harvest', detail: '4 parallel public-source harvesters' },
    { title: 'Synthesize', detail: 'merge findings into api-surface/ artifacts' },
    { title: 'Verify', detail: 'adversarial evidence check + completeness critic' },
    { title: 'Fix', detail: 'bounded fix loop (max 2 rounds)' },
  ],
}

const { date, repo, scratch } = args
const API = repo + '/api-surface'

const HARVEST_SCHEMA = {
  type: 'object',
  required: ['summary', 'findings_file', 'endpoint_candidates', 'warnings'],
  properties: {
    summary: { type: 'string', description: 'What was found, 3-6 sentences' },
    findings_file: { type: 'string', description: 'Absolute path of the findings YAML written' },
    endpoint_candidates: { type: 'number' },
    warnings: { type: 'array', items: { type: 'string' } },
  },
}

const SYNTH_SCHEMA = {
  type: 'object',
  required: ['files_written', 'endpoints_total', 'confirmed', 'inferred', 'unknown', 'notes'],
  properties: {
    files_written: { type: 'array', items: { type: 'string' } },
    endpoints_total: { type: 'number' },
    confirmed: { type: 'number' },
    inferred: { type: 'number' },
    unknown: { type: 'number' },
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
        required: ['file', 'description', 'severity'],
        properties: {
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

const COMMON = `
You are one of four parallel harvesters gathering PUBLIC evidence about the Anvilogic SaaS platform API (SOC detection-engineering platform, control plane at secure.anvilogic.com, auth = static API key as Authorization: Bearer). Your findings feed a synthesis agent that builds an endpoint registry.

Write your findings as ONE YAML file (create the directory if needed) and return the structured summary. Findings YAML shape:
  source_urls: [{url, fetched: '${date}', access: public, note}]
  auth_facts: [{fact, evidence_url}]
  endpoint_candidates: [{tentative_id, domain, method (or UNKNOWN), path (or UNKNOWN), summary, evidence_url, evidence_kind, confidence_suggestion: inferred|unknown, notes}]
  domain_facts: [{noun, definition, relationships, evidence_url}]
  convention_facts: [{topic: pagination|rate-limit|errors|versioning|hosts, fact, evidence_url}]
  warnings: [string]

Rules: NEVER invent endpoint paths — if a source only names an action (e.g. a SOAR connector 'List Detections'), record method/path as UNKNOWN with confidence_suggestion inferred. Record every URL you actually fetched. Domains vocabulary: detections, scenarios, eoi, hunting, alerts, feeds, forge, blueprints, metrics, platform. If a fetch fails or is blocked, note it in warnings and move on (max 1 retry per URL).`

const harvesters = [
  {
    key: 'public-docs',
    prompt: `${COMMON}
YOUR SOURCE: the public Anvilogic docs GitBook at https://public-docs.anvilogic.com/. Fetch https://public-docs.anvilogic.com/llms.txt and https://public-docs.anvilogic.com/sitemap.md first to map the site, then fetch (WebFetch) the pages most relevant to: platform connectivity (secure.anvilogic.com, eoi-files.anvilogic.com, databus.anvilogic.com), API key generation (Settings -> Generate API Key), SIEM app configuration, EOI routing, data repository onboarding (Splunk/Snowflake/Databricks/Azure), and any mention of REST endpoints, API paths, tokens, rate limits, or pagination. https://public-docs.anvilogic.com/llms-full.txt is the full corpus — fetch it if the individual pages are insufficient, but prefer targeted pages. Aim for breadth across ALL platform domains, not just onboarding.
Write findings to ${scratch}/findings-public-docs.yaml`,
  },
  {
    key: 'armory',
    prompt: `${COMMON}
YOUR SOURCE: github.com/anvilogic-forge/armory (public repo of Anvilogic detection content, GPL-3.0). Use read-only gh api calls or a shallow clone into ${scratch}/armory-clone. Mine the YAML detection files: catalog the schema (field names, types, MITRE ATT&CK mapping structure, data-source/macro references, threat-identifier vs threat-scenario structure if both exist), directory taxonomy, and any metadata that reveals platform object models (deployment state, repo targets, versioning). These are payload-shape and domain-model facts — endpoint_candidates only if a file literally references an API path. Also check the repo README/docs for API or platform references.
Write findings to ${scratch}/findings-armory.yaml`,
  },
  {
    key: 'sentinel',
    prompt: `${COMMON}
YOUR SOURCE: the Anvilogic solution inside github.com/Azure/Azure-Sentinel, path 'Solutions/Anvilogic/' (especially 'Data Connectors/AnviLogic_CCF/' files: Anvilogic_PollingConfig.json, Anvilogic_DataConnectorDefinition.json, Anvilogic_DCR.json, Anvilogic_Table.json). Fetch via gh api with raw accept header. IMPORTANT FRAMING: this connector queries the CUSTOMER'S Azure Data Explorer (POST <avl_adx_uri>/v2/rest/query, db=anvilogic, Azure AD OAuth) — it is NOT the Anvilogic SaaS control-plane API. Your findings feed the domain model (table schemas, alert/EOI field names, event shapes) and conventions context ONLY. Every endpoint_candidate you emit (if any) must carry evidence_kind 'adx-connector' and a warning that it is ADX-side, so the synthesizer keeps it OUT of the SaaS registry.
Write findings to ${scratch}/findings-sentinel.yaml`,
  },
  {
    key: 'soar-sweep',
    prompt: `${COMMON}
YOUR SOURCE: public SOAR/automation connector documentation for Anvilogic. Sweep with WebSearch + WebFetch: Tines (https://www.tines.com/docs/credentials/connect-flows/anvilogic/ and any Tines story-library pages listing Anvilogic actions), Torq, Palo Alto Cortex XSOAR marketplace, Splunk SOAR/Phantom app listings, Splunkbase app 4975 docs, and GitHub code search for 'secure.anvilogic.com' or 'anvilogic' API client code in public repos (gh api search/code, or the web UI search). Goal: action names and any leaked endpoint/header/auth details -> inferred endpoint candidates. Consumer-marketing pages may block fetches; per the fallback discipline retry a blocked URL at most once then move on and note it.
Write findings to ${scratch}/findings-soar.yaml`,
  },
]

phase('Harvest')
log('Fanning out 4 public-source harvesters')
// Barrier justified: synthesis needs ALL findings together to dedupe into one registry.
const results = await parallel(
  harvesters.map((h) => () => agent(h.prompt, { label: `harvest:${h.key}`, phase: 'Harvest', schema: HARVEST_SCHEMA }))
)
const findings = results.filter(Boolean)
if (findings.length === 0) throw new Error('All harvesters failed — aborting before synthesis')
log(`Harvest done: ${findings.length}/4 harvesters returned, ${findings.reduce((n, f) => n + f.endpoint_candidates, 0)} endpoint candidates`)

phase('Synthesize')
const synth = await agent(
  `Build the Wave-1 Anvilogic api-surface knowledge base from harvested findings.

INPUT: read these findings YAML files (some may be missing if a harvester failed):
${harvesters.map((h) => `- ${scratch}/findings-${h.key === 'public-docs' ? 'public-docs' : h.key === 'soar-sweep' ? 'soar' : h.key}.yaml`).join('\n')}

Harvester summaries:
${findings.map((f) => `--- ${f.findings_file}: ${f.summary} Warnings: ${f.warnings.join('; ') || 'none'}`).join('\n')}

OUTPUT: write these 6 files under ${API}/ :

1. endpoints.yaml — the registry. Header keys: version: 1, base_url: https://secure.anvilogic.com (mark INFERRED — exact API base path unknown), generated: '${date}'. Then endpoints: map keyed by stable id (e.g. detections.list). Each entry: domain (detections|scenarios|eoi|hunting|alerts|feeds|forge|blueprints|metrics|platform), method (GET/POST/... or UNKNOWN), path (string or UNKNOWN), summary, params (list, may be empty), pagination: {style: none|offset|page|cursor|link-header|unknown}, auth: bearer-api-key, risk: read|write|bulk|destructive, confidence: confirmed|inferred|unknown, evidence: list of {source (URL or sources.md anchor), kind: public-docs|repo|connector-page|adx-connector|gated-docs|live-response, note}, first_seen: '${date}', last_verified: null. RULES: confidence confirmed ONLY for facts directly evidenced by a public source showing the actual SaaS REST surface (rare in Wave 1 — likely only auth/host facts, which belong in auth.md not endpoints). SOAR action names -> inferred with method/path UNKNOWN unless the source shows them. NOTHING with kind adx-connector may appear as a SaaS endpoint — ADX material informs domain-model.md only. Do not invent paths.

2. domain-model.md — noun glossary + relationships (threat identifiers -> threat scenarios -> EOI -> alerts; data feeds/repositories per platform Splunk/Snowflake/Databricks/Azure; Forge/Armory content; blueprints; MTTD/maturity metrics), enriched with armory schema facts (detection YAML fields) and sentinel table shapes (clearly labeled as ADX-side observations).

3. auth.md — confirmed: static API key as Authorization Bearer, generated in UI Settings -> Generate API Key (admin role), host secure.anvilogic.com, TLS 1.2+. Flag UNKNOWN-until-Wave-3: exact header name confirmation, key expiry/rotation, API base path, scoping/permissions.

4. conventions.md — sections Pagination / Rate limits / Error envelope / Versioning / Hosts, EACH bannered with status: UNKNOWN|INFERRED|CONFIRMED. Hosts section can cite secure.anvilogic.com, eoi-files.anvilogic.com, databus.anvilogic.com facts.

5. coverage.yaml — matrix: for EVERY domain noun above, rows domain.operation (use CRUD-ish operations you'd expect: list/get/create/update/delete/search/deploy as sensible per domain) with status: captured|implemented|tested|documented booleans (Wave 1: captured true only where an endpoints.yaml entry exists; all others all-false rows so gaps are visible), and registry_id: (id or null).

6. sources.md — provenance table: every URL any harvester fetched: URL, fetch date, access class (public), what it evidenced, license/ToS note (e.g. armory GPL-3.0; Azure-Sentinel MIT; docs (c) Anvilogic — facts extracted, no verbatim republication).

Use clean YAML (no tabs). Prefer fewer, well-evidenced registry entries over speculative bulk. Return the structured summary.`,
  { label: 'synthesize:api-surface', phase: 'Synthesize', schema: SYNTH_SCHEMA }
)
if (!synth) throw new Error('Synthesis agent failed')
log(`Synthesized ${synth.endpoints_total} endpoints (${synth.confirmed} confirmed / ${synth.inferred} inferred / ${synth.unknown} unknown)`)

const verifierPrompt = `Adversarially verify the Anvilogic api-surface artifacts in ${API}/ (endpoints.yaml, auth.md, conventions.md, sources.md). You are trying to REFUTE claims, not confirm them.
Checks:
1. Evidence integrity: sample AT LEAST 8 endpoints.yaml entries (all of them if fewer) plus every 'confirmed' claim anywhere; re-fetch each cited evidence URL (WebFetch; 1 retry max) and check the source actually supports the claim. Unsupported/unreachable evidence -> issue (major; blocker if confidence would need downgrading).
2. Contamination: no entry with evidence kind adx-connector presented as a SaaS endpoint; no invented-looking paths (a concrete path whose evidence does not literally show that path is a blocker).
3. Confidence honesty: 'confirmed' requires direct public evidence of the SaaS REST surface; SOAR-action-derived entries must be 'inferred' with UNKNOWN method/path unless shown.
4. YAML validity: parse endpoints.yaml and coverage.yaml (python3 -c "import yaml,sys; yaml.safe_load(open(sys.argv[1]))" or equivalent); schema fields present on every entry.
Do NOT edit files. Return pass=true only with zero blocker/major issues.`

const criticPrompt = `Completeness-check the Anvilogic api-surface artifacts in ${API}/. Do NOT edit files.
1. Every domain noun (detections, scenarios, eoi, hunting, alerts, feeds, forge, blueprints, metrics, platform) appears in coverage.yaml, each with plausible expected operations — even as all-false rows.
2. Every endpoints.yaml entry has a matching coverage.yaml row (registry_id linkage) and vice versa where captured=true.
3. All 6 files exist and are non-trivial: endpoints.yaml, domain-model.md, auth.md, conventions.md, coverage.yaml, sources.md.
4. conventions.md has all 5 sections each with an explicit UNKNOWN/INFERRED/CONFIRMED banner.
5. sources.md covers the URLs mentioned in evidence entries (spot-check 10).
6. What is MISSING that Wave 3 (gated-docs capture) will need as anchors — flag as minor issues.
Return pass=true only with zero blocker/major issues.`

let clean = false
let remaining = []
for (let round = 0; round < 2; round++) {
  phase('Verify')
  const [ev, comp] = await parallel([
    () => agent(verifierPrompt, { label: `verify:evidence-r${round + 1}`, phase: 'Verify', schema: ISSUES_SCHEMA }),
    () => agent(criticPrompt, { label: `verify:completeness-r${round + 1}`, phase: 'Verify', schema: ISSUES_SCHEMA }),
  ])
  const issues = [...(ev ? ev.issues : []), ...(comp ? comp.issues : [])]
  const actionable = issues.filter((i) => i.severity !== 'minor')
  remaining = issues
  if (actionable.length === 0) {
    clean = true
    log(`Verification clean on round ${round + 1} (${issues.length} minor notes)`)
    break
  }
  log(`Round ${round + 1}: ${actionable.length} actionable issues — fixing`)
  await agent(
    `Fix these verified issues in the Anvilogic api-surface artifacts under ${API}/. Apply the upgrade rule: never delete endpoint ids (downgrade confidence / mark UNKNOWN instead), keep YAML valid, keep evidence citations accurate. Minor issues optional; fix all blocker/major.
Issues (JSON): ${JSON.stringify(actionable)}
Also listed minor notes for context: ${JSON.stringify(issues.filter((i) => i.severity === 'minor'))}
Return counts.`,
    { label: `fix:round${round + 1}`, phase: 'Fix', schema: FIX_SCHEMA }
  )
}

return {
  harvesters: findings.map((f) => ({ file: f.findings_file, candidates: f.endpoint_candidates, warnings: f.warnings })),
  synthesis: synth,
  verificationClean: clean,
  remainingIssues: remaining,
}