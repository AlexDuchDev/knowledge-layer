package chunks

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/platform/queue"
)

const defaultSourceType = "entity_body"
const maxChunkRunes = 900

// Chunk is a persisted retrievable fragment.
type Chunk struct {
	ID           uuid.UUID       `json:"id"`
	EntityID     uuid.UUID       `json:"entity_id"`
	SourceType   string          `json:"source_type"`
	TextContent  string          `json:"text_content"`
	Ordinal      int             `json:"ordinal"`
	TokenCount   int             `json:"token_count"`
	MetadataJSON json.RawMessage `json:"metadata_json"`
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
			INSERT INTO chunks (id, entity_id, source_type, text_content, ordinal, token_count, metadata_json)
			VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb)`,
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

// ListByEntity returns chunks ordered by ordinal.
func (s *Service) ListByEntity(ctx context.Context, entityID uuid.UUID) ([]Chunk, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, entity_id, source_type, text_content, ordinal, token_count, metadata_json
		FROM chunks WHERE entity_id = $1 ORDER BY ordinal ASC`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.EntityID, &c.SourceType, &c.TextContent, &c.Ordinal, &c.TokenCount, &c.MetadataJSON); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// Get returns one chunk by id.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Chunk, error) {
	var c Chunk
	err := s.pool.QueryRow(ctx, `
		SELECT id, entity_id, source_type, text_content, ordinal, token_count, metadata_json
		FROM chunks WHERE id = $1`, id,
	).Scan(&c.ID, &c.EntityID, &c.SourceType, &c.TextContent, &c.Ordinal, &c.TokenCount, &c.MetadataJSON)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
