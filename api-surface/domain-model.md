# Anvilogic Domain Model (Wave 1)

Synthesized 2026-07-06 from public docs (public-docs.anvilogic.com), the
anvilogic-forge/armory detection repo, the Azure-Sentinel Anvilogic solution,
and SOAR connector pages. Evidence provenance in `sources.md`.

## Core relationship chain

```
Forge (threat research team)
  └─> Armory (content catalog / marketplace)
        ├─> trending topics (in-product Forge Threat Detection Report)
        ├─> detection packs { threat identifiers + threat scenarios + macros }
        └─> threat identifiers (atomic detection rules)
              │  deployed to a data repository (Splunk / Snowflake / Databricks / Azure ADX)
              ▼
            EOIs (events of interest = detection output)
              ├─> EOI routing pipeline ──> primary EOI repository (hybrid setups)
              ├─> copy ──> platform Alert Lake (AI-Insights opt-in)
              ├─> correlated by threat scenarios (multi-stage sequencing)
              └─> alerts ──> SOAR (REST push or pull) / Sentinel incidents
```

## Nouns

### Threat identifier (detection rule)
An atomic, deployable detection rule. On Splunk, runs as a cron saved search in
the AVL app with results written into the Anvilogic index; on Snowflake, as
scheduled tasks in the Detect warehouse; on Databricks, as Workflow jobs (SQL
builder rules converted to PySpark). Building block of threat scenarios; member
of detection packs; produces EOIs. Platform rule handle: `avl_r<rule_id>`.

### Threat scenario / use case
Sequencing detection that strings together multiple threat identifiers,
deployed via the platform; correlation content built over EOIs (the hunting
index is used "to create Threat Scenario correlations"). Identified by
`AVL_UC<use_case_id>`; UI deep-link `https://secure.anvilogic.com/use_cases?id=AVL_UC1029`
(browser route, not a REST endpoint). No scenario files exist in the public
armory repo, so the scenario schema is unknown. Tines exposes a "create use
case" write action.

### Detection pack
A collection of threat identifiers, threat scenarios, and macros addressing a
specific security issue; viewable and deployable in the Armory.

### Trending topic
In-product version of the Forge Threat Detection Report emails; found on the
Home page and Armory; deployable as a bundle of content.

### Armory
Catalog surface showing all available detections not yet deployed in the
customer's system; also referenced as the integration marketplace. The public
GitHub repo `anvilogic-forge/armory` is "public versions of the sophisticated
detections found within the real Anvilogic Platform Armory".

### Forge
Anvilogic's threat-research team producing Armory content, trending topics, and
the Threat Detection Report. Contact forge@anvilogic.com.

### EOI (event of interest)
Fully normalized signal output by detections. Stored in the customer's
Anvilogic index (Splunk), alert table (Snowflake), or Anvilogic Alert Table /
`eoi` table (Azure ADX). Escalated to SOAR and used as a hunting index for
threat-scenario correlations. Alert-mode subset (ADX-side observation:
`avl_rule_mode != 'Warn'`) constitutes "Anvilogic Alerts".

### EOI routing pipeline
In hybrid multi-repository setups the customer selects a primary EOI data
repository; the pipeline routes all alerts, regardless of origin repo, to that
destination for cross-repository correlation. Anvilogic also stores a copy of
all alerts in the platform Alert Lake.

### Alert Lake
Platform-side store holding a copy of all alerts generated in the platform;
powers AI-Insights (Tuning, Health, Hunting escalations). Enrichment tables can
enrich detections before storage; sits upstream of SOAR.

### Alert data vs raw data
Raw data = events/telemetry from endpoints/tools/appliances; alert data =
curated vendor-judged signals (e.g. Proofpoint, Wiz, CrowdStrike). Vendor alert
integrations flow via `databus.anvilogic.com`; raw custom data flows via the
Anvilogic S3 ingestion pipeline (Snowflake only).

### Data feed
A categorized security data source in the customer repository; auto-categorized
and synchronized to the platform every 7 days. Has tags/data categories
(affecting MITRE coverage) and a feed-quality rating; only "Good"-quality feeds
drive recommendations. Auto-computed quality exists for Windows event logs in
Splunk.

### Data repositories (execution planes)
- **Splunk** — Anvilogic App for Splunk on the search head; detections as cron
  saved searches; EOIs to `<org>_anvilogic` index via a custom HEC collector
  command (`avl_hec_token`); paired `<org>_anvilogic_metrics` index for
  baselining output. Roles: avl_admin, avl_senior_developer, avl_developer,
  avl_senior_triage, avl_triage, avl_readonly; REST capabilities avl_get_rest,
  avl_post_rest, avl_post_rest_platform, avl_rest_config_access_get/post,
  avl_deploy_content, avl_write_hec.
- **Snowflake** — platform connects as `anvilogic_service` user with
  `anvilogic_admin` role (customer-run, platform-generated SQL); two warehouses
  (Ad-hoc, Detect); ingestion via Anvilogic-managed S3 pipeline.
- **Databricks** — detections run as Workflow jobs; SQL-builder rules converted
  to PySpark; SQL Warehouse + All-Purpose/Job compute; Lakeflow
  bronze/silver/gold ETL.
- **Azure (ADX/LA/Fabric)** — platform authenticates as a customer-created app
  service principal and queries via the Kusto `cluster()` command; Anvilogic
  provisions a per-tenant ADX cluster (`avl_adx_uri`) with database `anvilogic`
  and table `eoi` as the customer-facing EOI lake.

### Blueprints
Named only in the armory HEAD commit message ("Add JPEG architecture diagram
for Blueprints one-pager", 2026-03-31); confirms Blueprints is a current
Anvilogic offering/concept. Diagram assets in the repo are corrupted, so no
architectural detail is extractable. Structure UNKNOWN.

### Maturity Score
Real-time SOC maturity scoring giving continuous visibility into detection
posture mapped against priority threats. Contains Data Feeds; contributing
scores include the detection score (CSV import of existing rules) and feed
score. Related metric surface: "Retrieves detection coverages" (Tines action).

### Threat profile
Company profile (Region, Industry, Infrastructure) captured in guided
onboarding; filters applicable MITRE techniques so recommended content is
relevant; revisitable over time (platforms, threat groups, techniques, data
categories). Input to the recommendation engine alongside market/industry
trends, trusted group activity, popular search terms, and similar-org activity.

### Priorities (threat prioritization)
Org-scoped prioritization lists across threat groups, platforms, MITRE
techniques, and data categories, used to focus detection engineering. Exposed
via per-list read actions plus an "all defined priorities" action (Tines).

### Monte Copilot
Generative-AI copilot add-on (paid). Uses OpenAI-hosted models; PII stripped
before LLM processing; Q&A stored 30 days in an Anvilogic-owned AWS database.
Analyzes EOIs and entities/IOCs, returning a determination (MALICIOUS/BENIGN)
plus a report; SOAR surface is an async submit + retrieve-results pair, plus a
license-info action.

### Anvilogic App for Splunk
Bridges Splunk <-> secure.anvilogic.com / eoi-files.anvilogic.com /
databus.anvilogic.com using the static API key; runs detections, sends EOIs via
HEC, provides triage/allowlisting, health monitoring
(`| avlmanage command=check_app_health`).

## Armory detection YAML schema (repo evidence, GPL-3.0)

All 1,898 public detection files share exactly 9 top-level keys:

| Key | Meaning |
| --- | --- |
| `id` | Two-part `<use_case_id>.<rule_id>` (e.g. `1029.1032`); all logic variants of one detection share the use_case prefix; maps to platform handles `AVL_UC<use_case_id>` and `avl_r<rule_id>` |
| `title` | Detection title |
| `description` | Prose; often embeds flattened enrichment suffixes `-- Threat Actor Association: ...` and `-- Software Association: ...` (likely first-class fields in the platform model) |
| `logic_format` | `Splunk` (1,466 files) or `snowflake` (432 files) |
| `logic` | Query string (SPL or SQL) |
| `techniques` | List of `tactic:technique name` lowercase strings |
| `technique_id` | Parallel list of MITRE ATT&CK IDs (T####) |
| `data_category` | List of log-source categories (e.g. "EDR Logs", "Windows event logs", "AWS CloudTrail logs") |
| `references` | List of URLs or null |

Directory taxonomy: `detections/<category>/<name>/<name>-<logic_format>-<data_source>.yml`
with categories endpoint (499), cloud (122), authentication (69), web (49),
application (45), network (5), email (1) — 790 detection dirs.

Platform Splunk macro library observed in logic: `get_<domain>_data*` feed
routing macros (endpoint/cloud/web/authentication/application/network/email,
with per-source variants like `_winevent`, `_edr`, `_sysmon`, `_aws`, `_o365`),
`add_fields_eoi` (appends platform-standard EOI fields to output rows),
`hec_collect` (routes detection results to the HEC sink — the data-plane
alert-collection bridge), and `group_events("<fields>", <count-or-timespan>)`
(aggregation primitive). Snowflake-format logic is plain SQL over per-source
tables (crowdstrikefdr_process, awscloudtrail, okta, cloudflare_waf, gcpaudit,
snowflake.account_usage.query_history) with a ~2-hour lookback pattern.

Caveat: the public export is deliberately stripped (no deployment state, repo
targets, severity, timestamps) — absence of fields is not evidence about the
platform model.

## EOI record schema — ADX-SIDE OBSERVATIONS (label: adx-connector, NOT SaaS API evidence)

Source: Azure-Sentinel Anvilogic solution (custom table `Anvilogic_Alerts_CL`,
DCR, and PollingConfig). These describe the shape of EOI rows in the customer's
Anvilogic-provisioned ADX cluster (db `anvilogic`, table `eoi`) — the richest
public inventory of the Anvilogic alert/EOI object (289 columns) — but say
nothing about SaaS control-plane endpoints.

Field groups:

- **`avl_*` platform/detection metadata (~50 fields)** — tenancy `avl_org_id`
  (int, per-tenant); timing `avl_time`, `avl_event_time`; identity
  `avl_event_id`; rule fields `avl_rule_id/name/domain/sub_domain/mode/
  severity/risk_score(+_sum)/link`, logic text `avl_definition`, deploy/version
  tracking `avl_builder_version`, `avl_last_deployed_hash`; dedup/suppression
  `avl_duplicate_hash`, `avl_suppressed` (bool), `avl_suppression_key`;
  use-case fields `avl_use_case_id/name/title/type/category/sub_category/
  sub_type`; enrichment `avl_mitre_tactic/technique/ext_ids`,
  `avl_techniques_fqn`, `avl_threat_groups`, `avl_kill_chain_phase`,
  `avl_exploits`, `avl_vulnerabilities`, `avl_security_controls`,
  `avl_data_category`, `avl_custom_labels`; source labeling `avl_source`,
  `avl_sourcetype`, `avl_vendor_product/severity/risk_score`,
  `avl_victim_platform/product`; scenario fields `avl_stages`,
  `avl_stage_duration`, `avl_scenario_duration` (dynamic — an EOI can represent
  a multi-stage sequence); misc `avl_process_base64`, `avl_rest_id`,
  `avl_rest_name`.
- **`coi_*` extracted entities** ("candidates/context of interest"):
  `coi_account`, `coi_app`, `coi_domain`, `coi_host`, `coi_ip`, `coi_resource`,
  `coi_user` — used for Sentinel entity mapping.
- **Unprefixed normalized event fields** — Splunk-CIM-style (src_ip/dest_ip,
  ports, process_*/parent_process_*, file_*, registry_*, http_*, dns-ish
  query/answer, cve_*/cvss_*, email recipient/subject) plus `raw` (dynamic
  original event).
- **`orig_*` upstream references** — `orig_avl_event_id`, `orig_source`,
  `orig_sourcetype`, `orig_index` suggest EOIs can reference
  upstream/originating events (scenario chaining).

Behavioral conventions observed (ADX-side): `avl_rule_mode = 'Warn'` marks
non-alerting/test-mode rules, filtered out by both the connector KQL and the
Sentinel analytic rule; the analytic rule titles alerts
`{avl_rule_id} - {avl_use_case_title} - {avl_use_case_type}` and fires
per-result (AlertPerResult). Field semantics are inferred from connector usage,
not vendor documentation.

## Identifier conventions

| Handle | Meaning |
| --- | --- |
| `AVL_UC<number>` | Use case / threat scenario id (UI deep-link param) |
| `avl_r<number>` | Deployed rule id (also seen as `apply_al(avl_rNNNN)` wrapper in armory filenames) |
| `<use_case_id>.<rule_id>` | Two-part detection id in armory YAML |
| `avl_org_id` | Per-tenant org id on every EOI record (ADX-side) |
| `<org>_anvilogic`, `<org>_anvilogic_metrics` | Customer Splunk index names |
| `avl_adx_uri` | Per-tenant Anvilogic ADX cluster URI (Azure) |
