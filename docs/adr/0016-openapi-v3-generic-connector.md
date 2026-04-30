# ADR-0016: OpenAPI v3 generic connector

## Status

**Accepted (2026-04-30).** v0.6.0.

## Context

By v0.5.x KL ships 18 bespoke connectors. Each new vendor (ClickUp, Pipedrive, Airtable, …) is ~200–400 lines of boilerplate Go: HTTP client, pagination logic, JSON parsing, normalize-to-record-type. Most vendors expose a documented OpenAPI spec; that spec already encodes the URL, params, and response shape. Adding code per vendor is mechanical work that should belong to configuration, not source.

Hugr's data-source model treats an HTTP API + OpenAPI spec as a first-class data source. We adopt the spirit (one generic connector backed by per-feed config) without adopting Hugr's GraphQL surface — KL stays REST/connector-first.

## Decision

A single new connector type **`openapi_v3`** (id `20000000-0000-0000-0000-000000000014`) handles any REST API whose listing operation matches the v0.6.0 constraints below. Per-feed config (in `source_feeds.connector_config_json`) supplies:

- `openapi_url` (HTTPS or `http://localhost...`) — the spec's location.
- `list_path` (e.g. `/issues`) — the path within the spec to poll.
- `record_type` (closed enum mapped to `chunks/extract.go`'s 14 types).
- `item_mapping` — `{ canonical_field: jsonpath }` extracts each list item into the normalized payload.
- `auth.type` = `"bearer"` + `auth.token` (only auth scheme in v0.6.0).
- `pagination.{offset_param, limit_param, page_size, max_pages}` — offset/limit only in v0.6.0.

### Constraints (the "what's NOT in v0.6.0" list)

- **Pagination strategies other than offset/limit.** Cursor and link-header pagination are deferred. Operators with cursor-pagination APIs continue to use a bespoke connector or wait for v0.7+.
- **Auth schemes other than bearer.** No OAuth2 client_credentials, no API-key-in-query, no HMAC. The OAuth proxy from v0.5.0 is for inbound MCP, not outbound connector calls; the two paths stay separate to avoid coupling. Adding `oauth2` outbound is its own ADR.
- **JSONPath filter expressions** (`?(...)`). Strict mode in `ValidateJSONPathExpr` rejects them at activation. Operator-supplied expressions to extract fields should be static — `$.id`, `$.body.title`, etc. Anything more complex is a config smell.
- **Spec size > 5 MB.** Caps enforced in `FetchAndValidateSpec`. Bigger is almost always misconfiguration.
- **External `$refs` in spec.** `loader.IsExternalRefsAllowed = false` so an attacker can't point the parser at a malicious URL via a $ref. Local-only refs.
- **Record types outside `chunks/extract.go`'s 14-type set.** Closed enum. New types must extend the chunk extractor first; this connector's job is mapping vendor data into known shapes, not inventing new shapes.

### Validation pipeline

1. `Service.ValidateSourceFeed` (existing Phase 4.2.2 hook) routes activation through the adapter's `ValidateSourceFeedConfig`, which runs:
   1. `FeedConfig.Validate()` — schema + JSONPath strict-mode.
   2. `FetchAndValidateSpec(ctx, openapi_url, list_path)` — fetches the spec under a 5 MB cap, parses via kin-openapi, asserts the operator's `list_path` exists as a `GET` operation.
2. Failures bubble back to the source-feed creation flow; the feed never reaches `status='active'` with a bad config.

### Sync execution

`Run(ctx, baseURL, cfg, httpClient, onItem)` is the single entry. Per page:
- GET `{baseURL}{list_path}?{offset_param}={pageSize*pageN}&{limit_param}={pageSize}`.
- Bearer header set from `auth.token`.
- Decode JSON, apply `list_pointer` (default `$`) to find the items array.
- For each item, apply `item_mapping` JSONPaths → normalized payload (`record_type`, plus the mapped fields). Hash for dedup.
- Stop when fewer items returned than `page_size` (last page) or `max_pages` reached.

## Consequences

### Positive

- **New connectors land via config, not code.** Operator pastes an OpenAPI URL + 5 JSONPaths and has a working feed.
- **Closed `record_type` enum** keeps chunk extraction predictable — every item produced by an openapi_v3 feed flows through the same `chunks/extract.go` switch as the bespoke connectors.
- **Strict JSONPath** rejects expressions that could probe upstream beyond the documented mapping use case.
- **Spec size cap + no external $refs** keeps memory and SSRF surface bounded.

### Negative

- Operators with cursor-paginated APIs still need a bespoke connector. v0.7 is the planned widening.
- Operators with non-bearer auth (api-key in header, OAuth2 client_credentials) need a bespoke connector or a follow-up ADR.
- One config-mistake class — wrong JSONPath that maps to nothing — produces silently-empty items. We treat missing fields as empty (not error) so individual bad records don't break the sync; operator must `kltools schema-info` post-sync to confirm chunks landed.

### Neutral

- Per-feed token storage matches the existing connector pattern (notion, jira, etc.). Encryption-at-rest for `connector_config_json` secrets is a pre-existing gap not unique to this connector. A separate ADR will address blanket encryption when an operator demands it.

## Alternatives considered

- **Auto-derive `record_type` from spec semantics.** Rejected — too magical. We tried mapping by `path tags` or `operationId`, but every vendor names things differently. Explicit operator choice from a closed enum is honest.
- **Inline OpenAPI spec instead of fetching by URL.** Operators could paste the spec into config. Rejected: 5 MB pasted into a JSON column is awkward, and most operators want the connector to track the live spec (the URL is canonical).
- **Generic GraphQL connector.** Considered briefly because Hugr inspired the package. Rejected for v0.6.0 — GraphQL has no equivalent of "the spec encodes the URL"; the operator-config surface would balloon. v1.x consideration.

## Documentation updates

- `.env.example` and `docs/CONFIG_ENV.md` get an "OpenAPI v3 connector" note (no new env vars; everything is per-feed config).
- `docs/CONNECTOR_CAPABILITY_MATRIX.md` — add `openapi_v3` row.

## Revisiting

- **v0.7 — cursor pagination + link-header.** When operators ask for it.
- **v0.7 — OAuth2 client_credentials outbound.** Likely shares code with `connectoroauth/`; ADR follow-up.
- **v0.8 — auto-suggest `item_mapping` from spec response schema.** Uses kin-openapi to inspect the GET 200 response and propose JSONPaths for fields named `id`, `title`, `body`, etc.
