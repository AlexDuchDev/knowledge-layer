package secondbrain

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/platform/queue"
)

// ProcessOutboundDelivery sends a queued outbound message (Telegram when linked; Mattermost stub).
func ProcessOutboundDelivery(ctx context.Context, pool *pgxpool.Pool, telegramBotToken string, p queue.SecondBrainOutboundPayload) error {
	uid, err := uuid.Parse(strings.TrimSpace(p.UserID))
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(p.Channel)) {
	case "telegram":
		return SendTelegramToUser(ctx, pool, telegramBotToken, uid, p.Text)
	case "mattermost":
		// Requires Mattermost bot token + site URL; ingestion already covers MM reads.
		return nil
	default:
		return nil
	}
}
