# Anvilogic Authentication (Wave 1)

## CONFIRMED (public sources)

- **Static API key, generated in the platform UI.** Settings > Generate API Key
  on the Anvilogic platform (secure.anvilogic.com); the key is pasted into the
  Anvilogic App for Splunk under Settings > App Configuration > General
  Settings > API Settings > "API Key". This single static key is the app's
  credential for talking to the Anvilogic SaaS platform.
  (public-docs: connect-to-the-anvilogic-platform.md)
- **Admin role required to configure the key.** Editing the Splunk app
  configuration (where the API key is entered) requires the `avl_admin` Splunk
  role. (public-docs: connect-to-the-anvilogic-platform.md,
  assign-the-avl_admin-role.md)
- **Single-key, all-endpoints scope corroborated by Tines.** The Tines
  Anvilogic credential is a single "Text" credential whose only required input
  is an "API key", with scope described as "Allows access to all endpoints in
  the Anvilogic API". (tines.com/llm/docs/credentials/connect-flows/anvilogic.md)
- **Host and transport.** Control plane at `https://secure.anvilogic.com`;
  additional service hosts `eoi-files.anvilogic.com` and
  `databus.anvilogic.com`. All communication is REST over HTTPS/443 with
  TLS v1.2+. (public-docs: splunk-enterprise/verify-requirements.md,
  anvilogic-on-splunk-architecture.md)
- **REST capability model on the Splunk side.** App REST calls are gated by
  Splunk capabilities `avl_get_rest`, `avl_post_rest`, `avl_post_rest_platform`,
  `avl_rest_config_access_get/post`, `avl_deploy_content`, `avl_write_hec` —
  confirming GET and POST traffic to the platform with a distinct
  "post to platform" capability. (public-docs: assign-the-avl_admin-role.md)
- **Proxy support.** A proxy can be configured in the Anvilogic App for Splunk
  if the network requires one to reach Anvilogic.
- **Human auth is separate.** Platform UI login is email + password (welcome
  email), with Duo MFA and SSO configurable (details on the gated
  docs.anvilogic.com). Distinct from the machine API key.

## Adjacent auth planes (do NOT conflate with the control-plane key)

- **Azure data plane:** the Microsoft Sentinel connector authenticates with
  Azure AD OAuth2 client_credentials (Client ID/Secret from an "Anvilogic app
  registration"), token endpoint
  `https://login.microsoftonline.com/<tenant_id>/oauth2/v2.0/token`, scope
  `<avl_adx_uri>/.default`, and queries the customer's per-tenant ADX cluster —
  not secure.anvilogic.com.
- **Azure query path:** the platform itself queries customer ADX/LA/Fabric as a
  customer-created app service principal via the Kusto `cluster()` command.
- **Snowflake:** the platform connects as a dedicated `anvilogic_service`
  Snowflake user with an `anvilogic_admin` role (customer-run,
  platform-generated SQL); a Snowflake S3 storage integration grants access to
  an Anvilogic-managed bucket.
- **Forward-events (custom data):** per-integration S3 access key/id pairs
  issued by Anvilogic ("these change for each integration"), not the platform
  API key.

## UNKNOWN — resolve in Wave 3 (live/gated verification)

| Item | Status |
| --- | --- |
| Exact header name (`Authorization: Bearer <key>` vs `X-API-Key` etc.) | UNKNOWN — the Bearer form is asserted in the task brief but NOT confirmed by any public source; docs only show key generation + paste into the Splunk app field |
| Key expiry / rotation / revocation behavior | UNKNOWN |
| Multiple concurrent keys per org | UNKNOWN |
| API base path under secure.anvilogic.com (e.g. `/api`, `/v1`) | UNKNOWN |
| Key scoping / per-endpoint permissions | UNKNOWN — Tines describes a single all-endpoints scope, suggesting no granular scoping, but unverified |
| Whether eoi-files/databus hosts use the same API key | UNKNOWN |
