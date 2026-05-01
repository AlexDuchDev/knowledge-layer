# Manual upload — operator guide

The **manual** connector lets users (with manage-feed permission in a domain) drop arbitrary content into a governed source feed without writing connector code. Shipped in [v0.7.0](../../CHANGELOG.md).

> **Each user-created collection is one `source_feed`** against the `manual` connector type. All v0.4.0+ governance — sensitivity caps, scenario gating, audit, embeddings, AccessEvaluator-scoped retrieval — applies the same way as for chat / docs / meeting feeds.

---

## What's supported

| Upload kind | Endpoint | Extractor | Notes |
|---|---|---|---|
| Pasted text | `POST /api/manual/collections/:id/text` | None — body is the prose | Title auto-derived from the first line if not supplied |
| File | `POST /api/manual/collections/:id/file` (multipart) | PDF (`ledongthuc/pdf`), DOCX (`nguyenthenguyen/docx`), HTML, plain text, Markdown, CSV, JSON | 50 MiB max per file. Unknown MIME stored without text + warning |
| URL | `POST /api/manual/collections/:id/url` | HTML readability (same as `http_url` connector) | 4 MiB body cap on the upstream fetch |
| YouTube | `POST /api/manual/collections/:id/youtube` | Caption track via `kkdai/youtube/v2` (manual EN → manual any → auto-generated) | Missing tracks = metadata-only artifact + warning |

All four produce `docs_page`-shaped normalized records → chunks → embeddings via the v0.3.0 pipeline. Search and Ask see them automatically.

---

## Lifecycle

```
operator creates collection         POST /api/manual/collections
       ↓
collection = source_feed (active)   one row in source_feeds, type='manual'
       ↓
upload (text/file/url/youtube)      POST /api/manual/collections/:id/{kind}
       ↓
raw_artifact + normalized_record    inline; chunks/embeddings hook fires
       ↓
visible to /search and /ask         scoped by domain + sensitivity per AccessEvaluator
```

---

## Quick start (curl)

```bash
# 1. Create a collection in your domain.
curl -sS -X POST https://kl.example.com/api/manual/collections \
  -H 'Authorization: Bearer …' \
  -H 'Content-Type: application/json' \
  -d '{
    "label": "Q3 finance review notes",
    "description": "Manual brain-dump after the quarterly review",
    "domain_id": "32000000-0000-0000-0000-000000000001",
    "sensitivity_level": 1
  }'
# → {"feed_id": "...", "label": "...", "artifact_count": 0, ...}

# 2. Paste text.
curl -sS -X POST https://kl.example.com/api/manual/collections/$FEED_ID/text \
  -H 'Authorization: Bearer …' -H 'Content-Type: application/json' \
  -d '{"title":"Stripe billing migration plan", "body": "..."}'

# 3. Upload a PDF.
curl -sS -X POST https://kl.example.com/api/manual/collections/$FEED_ID/file \
  -H 'Authorization: Bearer …' \
  -F 'file=@/path/to/report.pdf'

# 4. Drop a URL.
curl -sS -X POST https://kl.example.com/api/manual/collections/$FEED_ID/url \
  -H 'Authorization: Bearer …' -H 'Content-Type: application/json' \
  -d '{"url":"https://stripe.com/blog/2024/some-post"}'

# 5. Pull a YouTube transcript.
curl -sS -X POST https://kl.example.com/api/manual/collections/$FEED_ID/youtube \
  -H 'Authorization: Bearer …' -H 'Content-Type: application/json' \
  -d '{"url":"https://www.youtube.com/watch?v=abcdefghijk"}'

# 6. Browse what landed.
curl -sS https://kl.example.com/api/manual/collections/$FEED_ID/artifacts \
  -H 'Authorization: Bearer …'
```

---

## In-collection search

For the "find in this collection" UX we ship a SQL `ILIKE` search keyed off the collection's chunks — fast, no embeddings dependency:

```bash
curl -sS -X POST https://kl.example.com/api/manual/collections/$FEED_ID/search \
  -H 'Content-Type: application/json' \
  -d '{"q":"stripe billing", "limit": 25}'
```

For broader semantic search, the global `/api/search` and `/api/ask` endpoints already cover manual content via the standard domain-permission flow.

---

## File extraction notes

| Format | Behaviour |
|---|---|
| PDF (text-based) | Extracted via per-page `GetPlainText`. Bad/encrypted pages are skipped with no error. |
| PDF (scanned, no OCR) | `manual_payload.warnings` includes `pdf contained no extractable text (possibly scanned without OCR)`. The artifact is still persisted; operator can renormalize after running OCR upstream. |
| DOCX | Path-based `nguyenthenguyen/docx` loader through a temp file. Strips XML tags down to prose. |
| HTML | `golang.org/x/net/html` walker; skips `<script>` and `<style>`. Captures `<title>`. |
| Markdown / CSV / JSON / plain text | Verbatim content, filename as title. |
| Image / video / archive / unknown | Stored without text + warning. Operator can still see and delete the artifact through the UI. |

---

## Renormalize

If extraction logic improves later, or the operator manually fixes `metadata_json`, force a rebuild:

```bash
curl -sS -X POST https://kl.example.com/api/manual/collections/$FEED_ID/artifacts/$ARTIFACT_ID/renormalize \
  -H 'Authorization: Bearer …'
```

This deletes the existing normalized_record (cascading to chunks + embeddings via FK) and recreates from the stored `metadata_json.manual_payload`. The hook re-fires; chunks and embeddings rebuild.

---

## Permissions

- **Create collection** — caller must have `source_feed` create permission in the target domain. Same gate as creating any other source feed.
- **Upload to collection** — caller must have `source_feed` manage permission for that feed. The upload routes call `requireManageSourceFeed` before the ingest method.
- **List / view** — restricted to domains the caller has at least view grant on. Uses `Access.DomainIDsWithGrant`.
- **Per-artifact delete** — same manage-feed gate; cross-feed deletes via id-guess are explicitly rejected.

---

## Audit events

Every operator action emits an `audit_events` row:

| Event type | Trigger |
|---|---|
| `manual_collection.created` | `POST /api/manual/collections` succeeded |
| `manual_collection.updated` | `PATCH /api/manual/collections/:id` |
| `manual_collection.archived` | `DELETE /api/manual/collections/:id` |
| `manual_artifact.uploaded.text` | text upload |
| `manual_artifact.uploaded.file` | file upload |
| `manual_artifact.uploaded.url` | URL upload |
| `manual_artifact.uploaded.youtube` | YouTube upload |
| `manual_artifact.deleted` | per-artifact delete |
| `manual_artifact.renormalized` | renormalize endpoint |

---

## Configuration

No new env vars. Tunables that affect manual uploads:

| Var | Default | What it affects |
|---|---|---|
| `BLOBSTORE_BACKEND` | unset | When set (`s3` etc.), file uploads also persist original bytes to the blob store; `raw_artifacts.storage_uri` filled. Without it, uploads still work but only the extracted text is retained. |
| `BLOBSTORE_S3_*` | — | Blob-store credentials. See [docs/CONFIG_ENV.md](../CONFIG_ENV.md). |

Constants in code:

- `MaxManualUploadSize` = **50 MiB** (per-file). Caller-side rejected at `POST .../file` with HTTP 413 before the ingest method runs.
- Body limit on the API server = **64 MiB** (`apps/api/cmd/api/main.go`). Accommodates the 50 MiB file plus multipart overhead.

---

## Common operator questions

**Q: Can a normal user create collections, or only admins?**
Anyone with `source_feed` create permission in a target domain. Operators typically grant this to power users via the existing role machinery. The collection's contents inherit the domain's sensitivity rules.

**Q: A user uploaded the same PDF twice. What happens?**
Second upload returns HTTP 200 with `deduped: true` and the original artifact's id. Dedup is on `content_hash` per feed.

**Q: Can I delete an entire collection's contents but keep the collection?**
Not in v0.7.0. Delete artifacts one at a time via `DELETE /api/manual/collections/:id/artifacts/:artifactId`, or archive the whole collection with `DELETE /api/manual/collections/:id` and create a new one.

**Q: Are file bytes stored anywhere if `BLOBSTORE_BACKEND` is unset?**
No. Without a blob store, the extracted text lives in `metadata_json.manual_payload.content_text`; the original bytes are not retained. This is the documented trade-off for operators who don't want to set up object storage.

**Q: A YouTube video has no captions — does upload fail?**
No. The artifact is persisted with metadata + a warning (`youtube video has no caption tracks; storing metadata only`). The user can see it in the collection but it has no extractable prose.

---

## Related docs

- [`docs/CONNECTOR_CAPABILITY_MATRIX.md`](../CONNECTOR_CAPABILITY_MATRIX.md) — `manual` row + comparison to other connectors
- [`docs/operations/kltools.md`](kltools.md) — `kltools schema-info` shows artifact counts per stage
- [`docs/operations/mcp.md`](mcp.md) — manual content is reachable through `kl_search` and `kl_ask_global`
- `apps/api/internal/ingestion_connectors/manual.go` — service surface
- `apps/api/internal/ingestion_connectors/manual_extract.go` — extractors
- `apps/api/internal/ingestion_connectors/manual_youtube.go` — YouTube captions
- `apps/api/internal/httpserver/manual_routes.go` — HTTP routes
