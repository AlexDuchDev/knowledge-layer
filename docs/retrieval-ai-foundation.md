# Retrieval and AI foundation

This note describes the first-class retrieval layer added alongside existing keyword search and Q&A. It complements [AI_RETRIEVAL_GOVERNANCE.md](./AI_RETRIEVAL_GOVERNANCE.md) with implementation-specific detail.

## Conceptual model

- **Knowledge chunks** (`chunks`): ordered fragments derived from entity `title` / `summary` / `body`, scoped to an entity. Chunks inherit the entity’s access rules; there is no separate chunk-level ACL in v1.
- **Embeddings** (`embeddings`): one row per `(chunk_id, model)` with a `vector(1536)` and optional `model_version`.
- **Keyword search** (`internal/search`): unchanged contract—OpenSearch or SQL title match, domain grants in SQL, then `permissions.Resolver.Evaluate(view)` per hit.
- **Entity graph (explore):** `GET /entities/:id/related` lists linked entities with per-target `view` checks; optional `depth=2` for bounded 2-hop expansion ([API_SURFACE_V1.md](./API_SURFACE_V1.md) §9.2). This is navigation/exploration, not a substitute for search index ACLs.
- **Semantic retrieval** (`internal/embeddings`): k-nearest neighbors on `embeddings` joined to `chunks` and `entities`, restricted by granted domain IDs in SQL, then the same `Evaluate(view)` second stage per candidate entity as search uses.
- **Hybrid retrieval** (`internal/retrieval`): fuses normalized keyword ranks and semantic similarity, minus a governance penalty derived from truth mode, lifecycle, freshness, and **approval_status** (approved preferred over non-approved).
- **Ask** (`internal/qa`): may consume ordered **context pieces** (entity + optional chunk text) produced by retrieval, while citations remain **entity-scoped** in the model JSON. Chunk IDs are stored on the answer trace for audit.

Raw connector artifacts are **not** indexed semantically in this increment; chunks are entity-scoped only until connector policy integration defines `view_raw` and feed rules.

## Permission enforcement (explicit)

1. **Search:** domain-scoped SQL, then `filterHitsByEntityView`.
2. **Semantic:** domain-scoped kNN, then `Evaluate(view)` per distinct entity on the candidate path.
3. **Hybrid:** only entities that pass keyword and/or semantic permission filtering participate in fusion; keyword hits are already filtered before scores are combined.
4. **Ask:** every entity loaded for evidence must pass `canView` / `Evaluate`; chunk text is never used without a passing entity check.

## HTTP: global ask

`POST /ask` accepts optional `retrieval_mode`:

| Value | Behavior |
|--------|----------|
| omitted / `keyword_only` | Permission-scoped keyword discovery (existing behavior), then synthesis. |
| `semantic_only` | Query embedding → semantic neighbors → chunk-aware synthesis. Requires embeddings in DB and `OPENAI_API_KEY`, `OPENROUTER_API_KEY`, or `OPENAI_MOCK=1`. |
| `hybrid` | Keyword hits + semantic candidates → fused ranking → chunk-aware synthesis where a semantic chunk exists for an entity. |

Weights: `RETRIEVAL_HYBRID_W_KEYWORD`, `RETRIEVAL_HYBRID_W_SEMANTIC`, `RETRIEVAL_HYBRID_PENALTY_WEIGHT` (defaults 0.45, 0.55, 0.02).

## Answer traces

Migration `000022` adds:

- `retrieval_mode`, `supporting_chunks_json`, `metrics_json`, `prompt_version` on `answer_traces`.

Global ask persists retrieval metrics (hit counts, latency, mode). Entity-scoped ask uses `retrieval_mode: entity_scoped` with empty chunk list.

## Chunk pipeline and worker

- On entity **create** and **patch** (title/summary/body), `EntityRepo` invokes a hook that runs `chunks.Service.RebuildEntityChunks` and, when `REDIS_URL` is set, enqueues `retrieval:embed_chunk` per new chunk.
- **`cmd/connectorworker`** handles `retrieval:embed_chunk`: loads chunk text, calls OpenAI embeddings (or mock), upserts `embeddings`.

Embedding model: `OPENAI_EMBEDDING_MODEL` or default `text-embedding-3-small` (must match stored dimensions).

## Schema

- `chunks`: `token_count`, `metadata_json`
- `embeddings`: `model_version`
- `answer_traces`: retrieval audit columns
- Index: `idx_embeddings_ivfflat_cosine` on `embeddings.embedding` (IVFFlat cosine; tune for production volume)

## Expansion (non-goals of this slice)

- Snippet/title leakage audit for OpenSearch pre-filter responses.
- HNSW tuning and per-model dimension configuration.
- Chunking strategies per `entity_type`.
- Normalized_record / raw artifact chunks behind explicit policies.
