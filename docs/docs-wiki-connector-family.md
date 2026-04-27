# Docs / wiki connector family

Shared **normalized page** shape for Notion, Google Docs exports, Confluence-style sources, and other hierarchical documentation.

## Normalized record

| `record_type` | Payload |
|---------------|---------|
| `docs_page` | `NormalizedDocPage` |

Fields include: `source_feed_id`, `connector_family` (`docs_wiki`), `connector_type`, `title`, `external_ref`, optional `external_doc_id`, `parent_ref` / `parent_refs`, `space_ref`, `owner_ref` / `editor_ref`, `labels`, `last_modified_at`, `mime_type`, `export_mime`, `body_text`, `web_view_link`, `downstream_hint`.

## Raw artifacts

- `docs_page`, `docs_page_revision` (when revisions are tracked)
- Confluence: `confluence_page`, `confluence_page_body`, `confluence_page_metadata`, `confluence_child_pages`

## Google Drive alignment

Google **Docs** files (`application/vnd.google-apps.document`) are normalized as `docs_page` via `docs_wiki.FromGoogleDriveExport`. Other Drive file types continue to use `google_drive_document` with the legacy structured payload for backward compatibility.

## Notion v1

- `connector_config_json`: `notion_integration_token`, `scope` (`page` or `database`).
- `external_ref`: Notion page UUID or database UUID.
- Database scope: shallow query (first 25 pages per sync) with per-page block text extraction.

## Confluence v1 (Cloud REST)

- `connector_config_json`: `confluence_base_url` (e.g. `https://your-domain.atlassian.net/wiki`), `confluence_auth` (PAT or OAuth bearer token), `confluence_feed_kind`: `space` | `page_collection` | `content_tree`.
- `external_ref`: space key (`space`), comma-separated page IDs (`page_collection`), or root page ID (`content_tree`).
- Sync stores separate raw rows for page JSON, body storage, metadata/labels, and (in tree mode) child page listings; normalized row uses `connector_type` `confluence` and `record_type` `docs_page`.

## Package

Go: `internal/ingestion_connectors/families/docs_wiki`.
