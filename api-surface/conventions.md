# Anvilogic API Conventions (Wave 1)

Each section is bannered with its evidence status. "The public corpus" = the
full 40-page public docs GitBook plus armory repo, Azure-Sentinel solution, and
SOAR connector pages harvested 2026-07-06.

## Pagination — status: UNKNOWN

No pagination conventions (offset, page, cursor, link-header) are documented
anywhere in the public corpus. The only windowing observed is the Azure
Sentinel connector's time-window polling (`{_QueryWindowStartTime}` ..
`{_QueryWindowEndTime}` against `ingestion_time()`, packing all rows into one
list via KQL) — that is a Microsoft CCF/ADX pattern and MUST NOT be read as an
Anvilogic SaaS convention.

## Rate limits — status: UNKNOWN

No Anvilogic API rate limits are documented. The only throttling mention in the
corpus is Azure Data Explorer cluster search concurrency — an Azure-side limit,
not the Anvilogic API.

## Error envelope — status: UNKNOWN

No API error format or status-code conventions are documented anywhere in the
public corpus. (ADX-side errors follow Azure Data Explorer REST semantics —
irrelevant to the SaaS surface.)

## Versioning — status: UNKNOWN

No API version scheme, versioned path (`/v1` etc.), or version header appears
anywhere in the public docs. The `/v2/` in the Sentinel connector's
`<avl_adx_uri>/v2/rest/query` is Azure Data Explorer REST versioning (v2 frame
protocol with TableKind/Rows envelope) — NOT an Anvilogic version signal.

Object-identity versioning signals that ARE observed (INFERRED, armory repo):
detection id `<use_case_id>.<rule_id>` with per-variant rule ids; ADX-side EOI
fields `avl_builder_version` and `avl_last_deployed_hash` suggest rule
deploy/version tracking inside the platform model.

## Hosts — status: CONFIRMED (hostnames/transport); purposes partly INFERRED

CONFIRMED by public docs (Splunk Enterprise verify-requirements page; all
HTTPS/443, TLS v1.2+, described as "REST API"):

| Host | Documented purpose |
| --- | --- |
| `secure.anvilogic.com` | Control plane / platform UI; "required to download Splunk code and rules metadata". Also confirmed as control-plane host by armory UI deep-links (`/use_cases?id=AVL_UC####` — a browser route, not REST evidence). |
| `eoi-files.anvilogic.com` | Connectivity required; purpose NOT documented (INFERRED: EOI file exchange for the Alert Lake / AI-Insights copy). |
| `databus.anvilogic.com` | "To send events for third party vendor alert integrations." |

Related host facts:

- Platform-to-data-repo transport is uniformly "REST API using HTTPS/443 with
  TLS v1.2+" (stated for Splunk and Snowflake architectures). CONFIRMED.
- SOAR integration is "via REST API through either a push or a pull method".
  CONFIRMED as a statement; the surface behind it is UNKNOWN.
- Per-tenant Azure data plane: `<avl_adx_uri>` ADX cluster, database
  `anvilogic`, table `eoi` — a SEPARATE plane from the control plane, with
  separate OAuth2 auth. CONFIRMED (Sentinel connector), excluded from the SaaS
  registry.
- No `api.*` host and no REST paths appear anywhere in the public corpus.
- Docs-site convention (meta): appending `.md` to any public-docs.anvilogic.com
  URL returns raw markdown; deeper product docs are gated at docs.anvilogic.com
  (GitBook origin kevin-hwang.gitbook.io/welcome-to-anvilogic).

## Async patterns — status: INFERRED

Monte Copilot SOAR actions come in submit + retrieve-results pairs (EOI
analysis and entity analysis), implying an async submit-then-poll pattern with
some handle/id returned by the submit. Inferred from Tines action names only.
