package answertrace

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Row struct {
	ID                     uuid.UUID       `json:"id"`
	PrincipalID            uuid.UUID       `json:"principal_id"`
	EntityID               uuid.UUID       `json:"entity_id"`
	Question               string          `json:"question"`
	Answer                 string          `json:"answer"`
	CitationsJSON          json.RawMessage `json:"citations_json"`
	SupportingEntitiesJSON json.RawMessage `json:"supporting_entities_json"`
	ScopeJSON              json.RawMessage `json:"scope_json"`
	Model                  string          `json:"model"`
	RetrievalMode          string          `json:"retrieval_mode,omitempty"`
	SupportingChunksJSON   json.RawMessage `json:"supporting_chunks_json"`
	MetricsJSON            json.RawMessage `json:"metrics_json"`
	PromptVersion          string          `json:"prompt_version,omitempty"`
	PrivacyJSON            json.RawMessage `json:"privacy_json,omitempty"`
	CreatedAt              time.Time       `json:"created_at"`
}

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) Insert(ctx context.Context, row Row) error {
	if row.CitationsJSON == nil {
		row.CitationsJSON = []byte("[]")
	}
	if row.SupportingEntitiesJSON == nil {
		row.SupportingEntitiesJSON = []byte("[]")
	}
	if row.ScopeJSON == nil {
		row.ScopeJSON = []byte("{}")
	}
	if row.SupportingChunksJSON == nil {
		row.SupportingChunksJSON = []byte("[]")
	}
	if row.MetricsJSON == nil {
		row.MetricsJSON = []byte("{}")
	}
	if row.PrivacyJSON == nil {
		row.PrivacyJSON = []byte("{}")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO answer_traces (id, principal_id, entity_id, question, answer, citations_json, supporting_entities_json, scope_json, model,
			retrieval_mode, supporting_chunks_json, metrics_json, prompt_version, privacy_json)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		row.ID, row.PrincipalID, row.EntityID, row.Question, row.Answer,
		row.CitationsJSON, row.SupportingEntitiesJSON, row.ScopeJSON, row.Model,
		row.RetrievalMode, row.SupportingChunksJSON, row.MetricsJSON, row.PromptVersion, row.PrivacyJSON)
	return err
}

// ListRecent returns newest traces (operator diagnostics).
func (r *Repo) ListRecent(ctx context.Context, limit int) ([]Row, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, principal_id, entity_id, question, answer, citations_json, supporting_entities_json, scope_json, model,
			retrieval_mode, supporting_chunks_json, metrics_json, prompt_version, privacy_json, created_at
		FROM answer_traces
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Row
	for rows.Next() {
		var row Row
		if err := rows.Scan(&row.ID, &row.PrincipalID, &row.EntityID, &row.Question, &row.Answer,
			&row.CitationsJSON, &row.SupportingEntitiesJSON, &row.ScopeJSON, &row.Model,
			&row.RetrievalMode, &row.SupportingChunksJSON, &row.MetricsJSON, &row.PromptVersion, &row.PrivacyJSON, &row.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (*Row, error) {
	var row Row
	err := r.pool.QueryRow(ctx, `
		SELECT id, principal_id, entity_id, question, answer, citations_json, supporting_entities_json, scope_json, model,
			retrieval_mode, supporting_chunks_json, metrics_json, prompt_version, privacy_json, created_at
		FROM answer_traces WHERE id=$1`, id,
	).Scan(&row.ID, &row.PrincipalID, &row.EntityID, &row.Question, &row.Answer,
		&row.CitationsJSON, &row.SupportingEntitiesJSON, &row.ScopeJSON, &row.Model,
		&row.RetrievalMode, &row.SupportingChunksJSON, &row.MetricsJSON, &row.PromptVersion, &row.PrivacyJSON, &row.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &row, nil
}
