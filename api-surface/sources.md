# Sources & Provenance (Wave 1)

Every URL fetched by the four Wave-1 harvesters (public-docs, armory, sentinel,
soar). All access was public (no authentication, no gated content fetched). All
fetches 2026-07-06 unless noted.

License/ToS summary:

- **public-docs.anvilogic.com / anvilogic.com** — (c) Anvilogic. Facts
  extracted; no verbatim republication of page content.
- **github.com/anvilogic-forge/armory** — GPL-3.0. Schema/statistics facts
  extracted; detection logic not republished.
- **github.com/Azure/Azure-Sentinel** — MIT. Config facts and field inventory
  extracted.
- **tines.com** — (c) Tines. Action names and credential-doc facts extracted;
  no verbatim republication.
- **raw.githubusercontent.com/oshezaf/sentinelninja** — community-generated
  docs, license unverified; used only to corroborate the MIT-licensed
  Azure-Sentinel table schema.

## Anvilogic public docs GitBook (harvester: public-docs)

Convention: appending `.md` to any page URL returns raw markdown. All 40 pages
listed in llms.txt were fetched (plus the two indexes).

| URL | Evidenced |
| --- | --- |
| https://public-docs.anvilogic.com/llms.txt | Site index (40 pages); absence-of-API-docs conventions baseline |
| https://public-docs.anvilogic.com/sitemap.md | Same 40-page listing |
| https://public-docs.anvilogic.com/welcome-to-anvilogic.md | Product positioning (AI SOC platform) |
| https://public-docs.anvilogic.com/get-started/onboarding-guide.md | Onboarding overview |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/log-in-and-set-your-password.md | Human auth: welcome email + password |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/define-your-companys-threat-profile.md | Threat profile noun; platform.update_threat_profile candidate |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in.md | Repository choice hub |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository.md | Splunk integration overview |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/download-and-install-the-anvilogic-app-for-splunk.md | App install hub |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/download-and-install-the-anvilogic-app-for-splunk/splunk-cloud-platform.md | Splunk Cloud path hub |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/download-and-install-the-anvilogic-app-for-splunk/splunk-cloud-platform/verify-requirements.md | Splunk Cloud reqs; HEC/443, IP allowlist |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/download-and-install-the-anvilogic-app-for-splunk/splunk-cloud-platform/install-the-anvilogic-app-for-splunk.md | Splunkbase install steps |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/download-and-install-the-anvilogic-app-for-splunk/splunk-enterprise.md | Splunk Enterprise path hub |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/download-and-install-the-anvilogic-app-for-splunk/splunk-enterprise/verify-requirements.md | KEY: three platform hosts (secure/eoi-files/databus.anvilogic.com) + purposes, HTTPS/443 |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/download-and-install-the-anvilogic-app-for-splunk/splunk-enterprise/download-the-anvilogic-app-for-splunk.md | platform.download_splunk_app candidate |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/download-and-install-the-anvilogic-app-for-splunk/splunk-enterprise/install-the-anvilogic-app-for-splunk.md | Install steps (no API detail) |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/create-the-anvilogic-indexes.md | `<org>_anvilogic(_metrics)` index model |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/assign-the-avl_admin-role.md | KEY: Splunk roles + avl_*_rest capabilities (GET/POST REST proof) |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/configure-the-hec-collector-commands.md | Splunk-side HEC (avl_hec_token), not Anvilogic API |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-splunk-as-your-data-repository/connect-to-the-anvilogic-platform.md | KEY: API key generation UI flow; avl_admin requirement; health check command; proxy support |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-snowflake-as-your-data-repository.md | Snowflake service user/role model; platform.integrate_snowflake candidate |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/select-your-data-repository-and-get-data-in/integrate-snowflake-as-your-data-repository/get-data-into-snowflake.md | Pipelines; references gated docs pages |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/review-data-feeds.md | Data feed noun; 7-day sync; feeds.update candidate |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/optional-upload-your-existing-detections.md | detections.import_existing candidate (CSV) |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/review-and-deploy-recommended-content.md | Content types; Armory; detections.deploy candidate |
| https://public-docs.anvilogic.com/get-started/onboarding-guide/additional-tasks.md | RBAC/MFA/SSO pointers to gated docs; docs-site host conventions |
| https://public-docs.anvilogic.com/get-started/reference-architectures.md | Architecture hub |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-splunk-architecture.md | KEY: REST HTTPS/443 TLS1.2+ FAQ; EOI definition; SOAR push/pull; AI-Insights Alert Lake copy |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-azure.md | Azure service-principal auth; Kusto cluster() querying |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-azure/azure-costs-estimates.md | ADX concurrency throttling (Azure-side) |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-azure/log-analytics-cross-tenant-search.md | Azure Lighthouse RBAC (no Anvilogic API detail) |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-snowflake-architecture.md | KEY: Snowflake REST FAQ; warehouses; Alert lake; alert-vs-raw data |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-snowflake-architecture/fluentbit.md | FluentBit templates hub |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-snowflake-architecture/fluentbit/linux-data.md | Forward-events per-integration S3 bucket/keys; feeds.create_forwarding_integration candidate |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-snowflake-architecture/fluentbit/syslog-data.md | Same pattern (syslog) |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-snowflake-architecture/fluentbit/windows-data.md | Same pattern (Windows) |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-snowflake-architecture/fluentd.md | Fluentd equivalents |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-databricks-architecture.md | Databricks execution plane (Workflow jobs, PySpark) |
| https://public-docs.anvilogic.com/get-started/reference-architectures/hybrid-anvilogic-on-splunk-and-snowflake-architecture.md | KEY: EOI routing pipeline; Alert Lake |
| https://public-docs.anvilogic.com/get-started/reference-architectures/hybrid-anvilogic-on-splunk-and-azure-architecture.md | Same EOI-routing FAQ |
| https://public-docs.anvilogic.com/security-controls/ai-security-controls.md | Generic AI security controls |
| https://public-docs.anvilogic.com/security-controls/monte-copilot-and-ai-privacy-and-controls.md | Monte Copilot privacy model (OpenAI-hosted, 30-day Q&A retention) |

## anvilogic-forge/armory (harvester: armory; GPL-3.0)

| URL | Evidenced |
| --- | --- |
| https://github.com/anvilogic-forge/armory | Shallow clone, HEAD 6677454: 1,898 detection YAMLs, 790 dirs, 7 categories; 9-key schema; macro library; Blueprints commit message |
| https://api.github.com/repos/anvilogic-forge/armory | Repo metadata: GPL-3.0, topics [detection, detection-engineering, snowflake, splunk, threat-hunting] |
| https://api.github.com/orgs/anvilogic-forge/repos | Org has two public repos (armory, aws-falcon-data-forwarder) |
| https://github.com/anvilogic-forge/armory/blob/main/README.md | Threat Identifier vs Threat Scenario model; macros as data-set placeholders; stripped public export |
| https://github.com/anvilogic-forge/armory/blob/main/detections/endpoint/wscript_cscript_execution/wscript_cscript_execution-splunk-winevent.yml | UI deep-link secure.anvilogic.com/use_cases?id=AVL_UC1029 (also fetched raw by the soar harvester) |
| https://github.com/anvilogic-forge/armory/blob/main/detections/cloud/o365_login_events/o365_login_events-splunk-apply_al(avl_r7397).yml | avl_r<rule_id> handle; group_events macro |

## Azure-Sentinel Anvilogic solution (harvester: sentinel; MIT)

All fetched raw via `gh api` (Accept: application/vnd.github.raw). ADX-side
evidence only — informs domain-model.md, excluded from the SaaS registry.

| URL | Evidenced |
| --- | --- |
| https://api.github.com/repos/Azure/Azure-Sentinel/contents/Solutions/Anvilogic | Solution directory listing |
| https://api.github.com/repos/Azure/Azure-Sentinel/contents/Solutions/Anvilogic/Data%20Connectors | Single connector dir (AnviLogic_CCF) |
| https://api.github.com/repos/Azure/Azure-Sentinel/contents/Solutions/Anvilogic/Data%20Connectors/AnviLogic_CCF | Connector file listing |
| https://github.com/Azure/Azure-Sentinel/blob/master/Solutions/Anvilogic/Data%20Connectors/AnviLogic_CCF/Anvilogic_PollingConfig.json | ADX POST <avl_adx_uri>/v2/rest/query; db=anvilogic; KQL over `eoi`; avl_rule_mode!='Warn' filter (also fetched by soar harvester) |
| https://github.com/Azure/Azure-Sentinel/blob/master/Solutions/Anvilogic/Data%20Connectors/AnviLogic_CCF/Anvilogic_DataConnectorDefinition.json | OAuth2 client_credentials setup; avl_adx_uri placeholders (also fetched by soar harvester) |
| https://github.com/Azure/Azure-Sentinel/blob/master/Solutions/Anvilogic/Data%20Connectors/AnviLogic_CCF/Anvilogic_DCR.json | DCR stream Custom-Anvilogic_Alerts_CL (289 columns, pass-through transform) |
| https://github.com/Azure/Azure-Sentinel/blob/master/Solutions/Anvilogic/Data%20Connectors/AnviLogic_CCF/Anvilogic_Table.json | Authoritative 289-column EOI field inventory (avl_*/coi_*/normalized/raw) |
| https://github.com/Azure/Azure-Sentinel/blob/master/Solutions/Anvilogic/Data/Solution_AnviLogic.json | Solution manifest v3.0.0; CCF dependency |
| https://github.com/Azure/Azure-Sentinel/blob/master/Solutions/Anvilogic/ReleaseNotes.md | Initial release 3.0.0, 2025-06-20 |
| https://github.com/Azure/Azure-Sentinel/blob/master/Solutions/Anvilogic/SolutionMetadata.json | offerId azure-sentinel-solution-anvilogic; category Security - Automation (SOAR) |
| https://github.com/Azure/Azure-Sentinel/blob/master/Solutions/Anvilogic/Analytic%20Rules/Anvilogic_Alerts.yaml | Alert titling, MITRE mapping, coi_* entity mappings, Warn filter |

## SOAR connector pages (harvester: soar)

| URL | Evidenced |
| --- | --- |
| https://www.tines.com/docs/credentials/connect-flows/anvilogic/ | Tines credential doc (JS-rendered; read via browser) |
| https://www.tines.com/llm/docs/credentials/connect-flows/anvilogic.md | Single "API key" text credential; scope "all endpoints in the Anvilogic API" |
| https://www.tines.com/solutions/products/anvilogic/ | 20 pre-built Anvilogic action templates — basis of all Tines-derived registry entries (no paths/hosts) |
| https://www.tines.com/library/stories/1251077/ | Story "Gather correlated Splunk searches and add to Anvilogic use cases" (story JSON with real URLs not public) |
| https://www.tines.com/library/stories/1313960/ | Story "AI Event Triage with Anvilogic Copilot" (MALICIOUS/BENIGN determination flow) |
| https://www.anvilogic.com/integrations | SOAR integrations asserted: Tines, Torq, Cortex XSOAR, Splunk SOAR, FortiSOAR (marketing) |
| https://www.anvilogic.com/integrations/cortex-xsoar | Marketing page; no actions/endpoints |
| https://public-docs.anvilogic.com/get-started/reference-architectures/anvilogic-on-splunk-architecture | Non-.md variant of the docs page above; REST/443/TLS1.2+, SOAR push/pull |
| https://raw.githubusercontent.com/oshezaf/sentinelninja/master/Solutions%20Docs/tables/anvilogic-alerts-cl.md | Community corroboration of the 289-column Anvilogic_Alerts_CL schema |

(The soar harvester also re-fetched Anvilogic_PollingConfig.json,
Anvilogic_DataConnectorDefinition.json, and the armory wscript_cscript file —
deduplicated into the tables above.)

## Referenced but NOT fetched (Wave 2/3 targets)

- Gated docs: https://docs.anvilogic.com (GitBook origin
  kevin-hwang.gitbook.io/welcome-to-anvilogic) — deploy-a-detection-pack,
  import-existing-rules, snowflake-data-ingestion, forward-events,
  cribl-stream, manage-users, authentication-settings, RBAC/SSO/MFA pages.
- https://public-docs.anvilogic.com/llms-full.txt — skipped (all 40 pages
  fetched individually; corpus fully covered).
- Splunkbase app 4975 (Anvilogic App for Splunk) package internals.
- https://github.com/anvilogic-forge/aws-falcon-data-forwarder — sibling
  ingestion repo, out of Wave-1 scope.
- Tines story JSON exports (require tenant login; would reveal real HTTP
  request URLs for the 20 actions).
