package presetcatalog

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LogRepository writes preset_instantiation_logs.
type LogRepository struct {
	pool *pgxpool.Pool
}

func NewLogRepository(pool *pgxpool.Pool) *LogRepository {
	return &LogRepository{pool: pool}
}

// Insert records an instantiation for audit.
func (r *LogRepository) Insert(ctx context.Context, presetID, principal uuid.UUID, targetKind string, targetID uuid.UUID, payload json.RawMessage) error {
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO preset_instantiation_logs (preset_catalog_entry_id, principal_user_id, target_kind, target_id, payload_json)
		VALUES ($1,$2,$3,$4,$5)`, presetID, principal, targetKind, targetID, payload)
	return err
}
