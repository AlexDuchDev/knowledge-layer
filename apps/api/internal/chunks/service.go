package chunks

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/platform/queue"
)

const defaultSourceType = SourceTypeEntityBody
const maxChunkRunes = 900

// Chunk is a persisted retrievable fragment. After migration 000041 (v0.3.0)
// exactly one of (EntityID, NormalizedRecordID) is non-zero; SourceType
// discriminates between entity-rooted and normalized_record-rooted chunks.
type Chunk struct {
	ID                 uuid.UUID       `json:"id"`
	EntityID           uuid.UUID       `json:"entity_id,omitempty"`
	NormalizedRecordID uuid.UUID       `json:"normalized_record_id,omitempty"`
	SourceType         string          `json:"source_type"`
	TextContent        string          `json:"text_content"`
	Ordinal            int             `json:"ordinal"`
	TokenCount         int             `json:"token_count"`
	MetadataJSON       json.RawMessage `json:"metadata_json"`
}

// Service manages entity chunking and schedules embedding jobs.
type Service struct {
	pool *pgxpool.Pool
	pub  *queue.Publisher
}

func NewService(pool *pgxpool.Pool, pub *queue.Publisher) *Service {
	return &Service{pool: pool, pub: pub}
}

// OnEntityPersisted rebuilds chunks for an entity and enqueues embedding tasks when Redis is enabled.
func (s *Service) OnEntityPersisted(ctx context.Context, entityID uuid.UUID) error {
	ids, err := s.RebuildEntityChunks(ctx, entityID)
	if err != nil {
		return err
	}
	if s.pub == nil || !s.pub.Enabled() {
		return nil
	}
	for _, id := range ids {
		if err := s.pub.EnqueueRetrievalEmbedChunk(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// RebuildEntityChunks replaces all chunks for the entity from title/summary/body. Returns new chunk IDs in ordinal order.
func (s *Service) RebuildEntityChunks(ctx context.Context, entityID uuid.UUID) ([]uuid.UUID, error) {
	var title string
	var summary, body *string
	err := s.pool.QueryRow(ctx, `
		SELECT title, summary, body FROM entities WHERE id = $1 AND archived_at IS NULL`, entityID,
	).Scan(&title, &summary, &body)
	if err != nil {
		return nil, err
	}
	parts := []string{strings.TrimSpace(title)}
	if summary != nil && strings.TrimSpace(*summary) != "" {
		parts = append(parts, strings.TrimSpace(*summary))
	}
	if body != nil && strings.TrimSpace(*body) != "" {
		parts = append(parts, strings.TrimSpace(*body))
	}
	full := strings.Join(parts, "\n\n")
	pieces := splitIntoChunks(full, maxChunkRunes)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM chunks WHERE entity_id = $1`, entityID)
	if err != nil {
		return nil, err
	}

	meta, _ := json.Marshal(map[string]string{"entity_id": entityID.String()})
	var ids []uuid.UUID
	for i, text := range pieces {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		id := uuid.New()
		tokens := estimateTokens(text)
		_, err = tx.Exec(ctx, `
			INSERT INTO chunks (id, entity_id, normalized_record_id, source_type, text_content, ordinal, token_count, metadata_json)
			VALUES ($1,$2,NULL,$3,$4,$5,$6,$7::jsonb)`,
			id, entityID, defaultSourceType, text, i, tokens, meta)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ids, nil
}

func estimateTokens(s string) int {
	n := utf8.RuneCountInString(s)
	if n <= 0 {
		return 0
	}
	return n / 4
}

func splitIntoChunks(s string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = maxChunkRunes
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	// Prefer paragraph boundaries
	paras := strings.Split(s, "\n\n")
	var cur strings.Builder
	curRunes := 0
	flush := func() {
		t := strings.TrimSpace(cur.String())
		if t != "" {
			out = append(out, t)
		}
		cur.Reset()
		curRunes = 0
	}
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		r := utf8.RuneCountInString(p)
		if curRunes+r > maxRunes && curRunes > 0 {
			flush()
		}
		if r > maxRunes {
			out = append(out, hardSplit(p, maxRunes)...)
			continue
		}
		if cur.Len() > 0 {
			cur.WriteString("\n\n")
			curRunes += 2
		}
		cur.WriteString(p)
		curRunes += r
	}
	flush()
	return out
}

func hardSplit(s string, maxRunes int) []string {
	var chunks []string
	runes := []rune(s)
	for len(runes) > 0 {
		if len(runes) <= maxRunes {
			chunks = append(chunks, string(runes))
			break
		}
		chunks = append(chunks, string(runes[:maxRunes]))
		runes = runes[maxRunes:]
	}
	return chunks
}

// ListByEntity returns entity-rooted chunks ordered by ordinal. Chunks rooted
// in normalized_records are excluded — use ListByNormalizedRecord for those.
func (s *Service) ListByEntity(ctx context.Context, entityID uuid.UUID) ([]Chunk, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, COALESCE(entity_id, '00000000-0000-0000-0000-000000000000'::uuid),
			COALESCE(normalized_record_id, '00000000-0000-0000-0000-000000000000'::uuid),
			source_type, text_content, ordinal, token_count, metadata_json
		FROM chunks WHERE entity_id = $1 ORDER BY ordinal ASC`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChunks(rows)
}

// ListByNormalizedRecord returns chunks that were extracted directly from a
// normalized_record (chat / docs / meeting / etc.) before any entity exists.
func (s *Service) ListByNormalizedRecord(ctx context.Context, normID uuid.UUID) ([]Chunk, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, COALESCE(entity_id, '00000000-0000-0000-0000-000000000000'::uuid),
			COALESCE(normalized_record_id, '00000000-0000-0000-0000-000000000000'::uuid),
			source_type, text_content, ordinal, token_count, metadata_json
		FROM chunks WHERE normalized_record_id = $1 ORDER BY ordinal ASC`, normID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChunks(rows)
}

// Get returns one chunk by id, regardless of source.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Chunk, error) {
	var c Chunk
	err := s.pool.QueryRow(ctx, `
		SELECT id, COALESCE(entity_id, '00000000-0000-0000-0000-000000000000'::uuid),
			COALESCE(normalized_record_id, '00000000-0000-0000-0000-000000000000'::uuid),
			source_type, text_content, ordinal, token_count, metadata_json
		FROM chunks WHERE id = $1`, id,
	).Scan(&c.ID, &c.EntityID, &c.NormalizedRecordID, &c.SourceType, &c.TextContent, &c.Ordinal, &c.TokenCount, &c.MetadataJSON)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanChunks(rows pgx.Rows) ([]Chunk, error) {
	var list []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.EntityID, &c.NormalizedRecordID, &c.SourceType, &c.TextContent, &c.Ordinal, &c.TokenCount, &c.MetadataJSON); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// RebuildNormalizedRecordChunks replaces all chunks for a normalized_record by
// extracting record-type-specific text via the per-type registry, splitting it
// into ≤ maxChunkRunes paragraph-aware fragments, and inserting them with
// source_type='normalized_record'. Returns new chunk IDs ordered by ordinal.
//
// Idempotent: re-running on the same normalized_record fully replaces the
// chunks (DELETE then INSERT in a transaction). Embedding tasks for the new
// chunks are enqueued when the publisher is enabled — same contract as
// RebuildEntityChunks.
//
// If the record_type has no registered extractor, returns nil, nil — caller
// is expected to skip silently. New record_types that arrive without a chunk
// extractor land in normalized_records and stay invisible to retrieval until
// extract.go's switch is updated. That trade-off is intentional: better to
// surface a follow-up issue than to embed garbage from an unknown shape.
func (s *Service) RebuildNormalizedRecordChunks(ctx context.Context, normID uuid.UUID) ([]uuid.UUID, error) {
	var recordType string
	var payload json.RawMessage
	err := s.pool.QueryRow(ctx, `
		SELECT record_type, structured_payload_json FROM normalized_records WHERE id = $1`, normID,
	).Scan(&recordType, &payload)
	if err != nil {
		return nil, err
	}

	text, ok := extractTextFromNormalizedRecord(recordType, payload)
	if !ok || strings.TrimSpace(text) == "" {
		// No extractor for this record_type, or the extractor returned empty.
		// Leave any existing chunks alone — DELETE+INSERT pattern would erase
		// chunks we may have produced previously under a different code path.
		return nil, nil
	}
	pieces := splitIntoChunks(text, maxChunkRunes)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM chunks WHERE normalized_record_id = $1`, normID)
	if err != nil {
		return nil, err
	}

	meta, _ := json.Marshal(map[string]string{
		"normalized_record_id": normID.String(),
		"record_type":          recordType,
	})
	var ids []uuid.UUID
	for i, piece := range pieces {
		piece = strings.TrimSpace(piece)
		if piece == "" {
			continue
		}
		id := uuid.New()
		tokens := estimateTokens(piece)
		_, err = tx.Exec(ctx, `
			INSERT INTO chunks (id, entity_id, normalized_record_id, source_type, text_content, ordinal, token_count, metadata_json)
			VALUES ($1, NULL, $2, $3, $4, $5, $6, $7::jsonb)`,
			id, normID, SourceTypeNormalizedRecord, piece, i, tokens, meta)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ids, nil
}

// OnNormalizedRecordPersisted rebuilds chunks for the record and enqueues
// embedding tasks. Mirror of OnEntityPersisted. Marks chunks_rebuilt_at on
// the source row so the periodic backfill (RebuildPendingNormalizedRecords)
// will not re-process it.
func (s *Service) OnNormalizedRecordPersisted(ctx context.Context, normID uuid.UUID) error {
	ids, err := s.RebuildNormalizedRecordChunks(ctx, normID)
	if err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE normalized_records SET chunks_rebuilt_at = now() WHERE id = $1`, normID); err != nil {
		return err
	}
	if s.pub == nil || !s.pub.Enabled() {
		return nil
	}
	for _, id := range ids {
		if err := s.pub.EnqueueRetrievalEmbedChunk(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// RebuildPendingNormalizedRecords drains up to `limit` normalized_records that
// have not yet been chunked (chunks_rebuilt_at IS NULL) and processes them
// in created_at order. Returns the number of records successfully processed.
//
// This is the central catch-up loop: connector adapters do `INSERT INTO
// normalized_records ...` at 24+ scattered sites, and rather than refactor
// every site to fire OnNormalizedRecordPersisted synchronously, the
// connectorworker's periodic task calls this method to converge the chunk
// surface eventually. Synchronous fire is reserved for high-fidelity callers
// (webhook_ingest path) that already route through Service.
//
// Lag: chunks-not-yet-built window equals the polling interval (default 30s
// in connectorworker). Acceptable for retrieval; "ingest then immediately
// ask" workflows can either route through PersistNormalizedRecord or trigger
// the backfill manually.
//
// Errors processing individual records do not abort the loop — they are
// surfaced via the returned slice of (normID, err) for the caller to log.
func (s *Service) RebuildPendingNormalizedRecords(ctx context.Context, limit int) (int, []error, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM normalized_records
		WHERE chunks_rebuilt_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return 0, nil, err
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}

	processed := 0
	var failures []error
	for _, id := range ids {
		if err := s.OnNormalizedRecordPersisted(ctx, id); err != nil {
			failures = append(failures, err)
			continue
		}
		processed++
	}
	return processed, failures, nil
}
