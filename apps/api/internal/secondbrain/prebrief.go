package secondbrain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/platform/queue"
)

// ProcessPreBriefTick sends due pre-meeting brief rows (polling path). Upstream jobs or calendar integration should INSERT into pre_meeting_brief_queue with dedupe_key.
func ProcessPreBriefTick(ctx context.Context, pool *pgxpool.Pool, pub *queue.Publisher) error {
	tgToken := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	for {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("secondbrain: prebrief begin: %w", err)
		}
		var id, userID, domainID uuid.UUID
		var payload []byte
		err = tx.QueryRow(ctx, `
			SELECT id, user_id, domain_id, payload_json
			FROM pre_meeting_brief_queue
			WHERE sent_at IS NULL AND scheduled_for <= now()
			ORDER BY scheduled_for ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED`).Scan(&id, &userID, &domainID, &payload)
		if err != nil {
			_ = tx.Rollback(ctx)
			if err == pgx.ErrNoRows {
				return nil
			}
			return fmt.Errorf("secondbrain: prebrief lock row: %w", err)
		}

		var meta map[string]any
		if len(payload) > 0 {
			_ = json.Unmarshal(payload, &meta)
		}
		title, _ := meta["title"].(string)
		if strings.TrimSpace(title) == "" {
			title = "Pre-meeting brief"
		}
		body := title + "\n\n" + formatBriefBody(meta)

		if pub != nil && pub.Enabled() {
			if err := pub.EnqueueSecondBrainOutbound(ctx, userID, "telegram", body); err != nil {
				_ = tx.Rollback(ctx)
				return fmt.Errorf("secondbrain: enqueue brief: %w", err)
			}
		} else if err := SendTelegramToUser(ctx, pool, tgToken, userID, body); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}

		if _, err := tx.Exec(ctx, `UPDATE pre_meeting_brief_queue SET sent_at = now() WHERE id=$1`, id); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("secondbrain: mark brief sent: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("secondbrain: prebrief commit: %w", err)
		}

		uid := userID
		did := domainID
		_ = RecordProductEvent(ctx, pool, "brief_sent", &uid, &did, nil, map[string]any{"queue_id": id.String()})
	}
}

func formatBriefBody(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	var parts []string
	if v, ok := meta["meeting_title"].(string); ok && strings.TrimSpace(v) != "" {
		parts = append(parts, "Meeting: "+strings.TrimSpace(v))
	}
	if v, ok := meta["starts_at"].(string); ok && strings.TrimSpace(v) != "" {
		parts = append(parts, "Starts: "+strings.TrimSpace(v))
	}
	if v, ok := meta["notes"].(string); ok && strings.TrimSpace(v) != "" {
		parts = append(parts, strings.TrimSpace(v))
	}
	return strings.Join(parts, "\n")
}
