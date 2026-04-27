package secondbrain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RecordProductEvent appends a row for OKR / analytics (BI-friendly).
func RecordProductEvent(ctx context.Context, pool *pgxpool.Pool, eventType string, actorUserID *uuid.UUID, domainID *uuid.UUID, extractedTaskID *uuid.UUID, payload map[string]any) error {
	if eventType == "" {
		return fmt.Errorf("secondbrain: event_type required")
	}
	var pj []byte
	if payload == nil {
		pj = []byte("{}")
	} else {
		var err error
		pj, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("secondbrain: payload json: %w", err)
		}
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO second_brain_product_events (id, event_type, actor_user_id, domain_id, extracted_task_id, payload_json)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5::jsonb)`,
		eventType, actorUserID, domainID, extractedTaskID, pj)
	if err != nil {
		return fmt.Errorf("secondbrain: insert product event: %w", err)
	}
	return nil
}

// ProductEvent is a row from second_brain_product_events for BI or in-app dashboards.
type ProductEvent struct {
	ID              uuid.UUID       `json:"id"`
	EventType       string          `json:"event_type"`
	ActorUserID     *uuid.UUID      `json:"actor_user_id,omitempty"`
	DomainID        *uuid.UUID      `json:"domain_id,omitempty"`
	ExtractedTaskID *uuid.UUID      `json:"extracted_task_id,omitempty"`
	PayloadJSON     json.RawMessage `json:"payload_json"`
	CreatedAt       time.Time       `json:"created_at"`
}

// ListProductEventsForDomain returns recent product events for a domain (newest first).
func ListProductEventsForDomain(ctx context.Context, pool *pgxpool.Pool, domainID uuid.UUID, limit int) ([]ProductEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
		SELECT id, event_type, actor_user_id, domain_id, extracted_task_id, payload_json, created_at
		FROM second_brain_product_events
		WHERE domain_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, domainID, limit)
	if err != nil {
		return nil, fmt.Errorf("secondbrain: list product events: %w", err)
	}
	defer rows.Close()
	var out []ProductEvent
	for rows.Next() {
		var e ProductEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.ActorUserID, &e.DomainID, &e.ExtractedTaskID, &e.PayloadJSON, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("secondbrain: scan product event: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
