# openapi_v3

Generic HTTP connector that polls a REST endpoint described by an OpenAPI 3.0/3.1 spec and maps each response item into a Knowledge Layer normalized_record via operator-supplied JSONPath. Shipped in v0.6.0.

Design rationale + scope cuts in [ADR-0016](../../../../../docs/adr/0016-openapi-v3-generic-connector.md). Capability matrix row in [docs/CONNECTOR_CAPABILITY_MATRIX.md](../../../../../docs/CONNECTOR_CAPABILITY_MATRIX.md).

## v0.6.0 constraints (intentional)

- **Pagination:** offset/limit only. Cursor + link-header → v0.7.
- **Auth:** bearer only. OAuth2 client_credentials → follow-up ADR.
- **JSONPath:** strict mode — `?(...)` filter expressions rejected at activation.
- **Spec size cap:** 5 MB.
- **External `$refs`:** disabled in the loader (SSRF defense).
- **`record_type`:** closed enum mapped to `chunks/extract.go`'s 14 known types.

## Files

- `config.go` — `FeedConfig` struct + `Validate()` covering 11 misconfiguration classes. Hardcoded `SupportedRecordTypes` map mirrors `chunks/extract.go`.
- `jsonpath.go` — `ValidateJSONPathExpr` rejects filter expressions at activation. `EvalString` returns empty string (not error) on missing field — individual bad records don't abort sync.
- `spec_validate.go` — `FetchAndValidateSpec(ctx, openAPIURL, listPath)` downloads with 5MB cap + 10s timeout, parses via `kin-openapi`, asserts `list_path` exists as a `GET` operation. `IsExternalRefsAllowed = false` blocks SSRF via $ref.
- `sync.go` — `Run(ctx, baseURL, cfg, httpClient, onItem)` paginates via offset/limit, applies `list_pointer` to find the items array, maps each item via `item_mapping`, hashes for dedup. Caller owns persistence (`Service.PersistNormalizedRecord`).
- `adapter.go` — `Adapter` implements `ingestion_connectors.ConnectorAdapter`. `ValidateSourceFeedConfig` runs `FeedConfig.Validate()` + `FetchAndValidateSpec` at activation. `SyncFeed` is a stub — actual sync is driven by `Service.SyncOpenAPIV3` (or queued connector:source_sync) which has access to `PersistNormalizedRecord`.
- `openapi_v3_test.go` — 21 sub-tests:
  - JSONPath strict-mode rejects 3 filter shapes (`$.items[?(@.id > 0)]`, etc.) and accepts 5 static paths.
  - 11 config-misuse cases (missing url, http non-localhost, missing list_path, unknown record_type, etc.).
  - Spec size cap enforced (6MB rejected).
  - Valid spec happy path + missing-list-path failure.
  - `EvalString` soft-empty on missing field.

## Per-feed config shape

Operator supplies via `source_feeds.connector_config_json`:

```json
{
  "openapi_url":  "https://api.vendor.com/openapi.json",
  "list_path":    "/issues",
  "record_type":  "work_item",
  "auth":         { "type": "bearer", "token": "..." },
  "pagination":   { "page_size": 50, "max_pages": 20,
                    "offset_param": "offset", "limit_param": "limit" },
  "item_mapping": {
    "external_ref": "$.id",
    "title":        "$.title",
    "description":  "$.body"
  },
  "list_pointer": "$.data"
}
```

`item_mapping.external_ref` is required (used for dedup across syncs). Other canonical fields depend on the chosen `record_type` — see [`chunks/extract.go`](../../../chunks/extract.go) for what each type's chunker reads.

## Integration with the rest of the connector framework

- Registered in `app/deps.go` alongside the other 19 adapters.
- `Service.ValidateSourceFeed` (existing Phase 4.2.2 hook) routes through `Adapter.ValidateSourceFeedConfig` at activation — bad config never becomes an active feed.
- Sync execution lives in `Service.SyncOpenAPIV3` (or a queued `connector:source_sync` task in the connectorworker). Each successfully mapped item flows through `Service.PersistNormalizedRecord` (the v0.3.0 helper) so the chunks-rebuild hook + audit emission happen uniformly.
- The configured `record_type` determines which `chunks/extract.go` switch case the resulting normalized_records flow through. Operator typo → adapter rejects at activation; nothing reaches an unmapped extractor.

## Adding cursor / link-header pagination (v0.7+)

Extend `PaginationConfig` with a discriminator (`type: "offset_limit" | "cursor" | "link_header"`). Branch in `sync.Run`. Add adversarial tests for the new code paths. Update ADR-0016 with the deprecation of "offset/limit only" wording.

## Adding OAuth2 client_credentials (follow-up ADR)

Extend `AuthConfig` with `oauth2_client_credentials` type and a token cache. The cache likely belongs in the service layer (refresh tokens cross sync runs), not the per-call adapter — design the contract first, then implement.
