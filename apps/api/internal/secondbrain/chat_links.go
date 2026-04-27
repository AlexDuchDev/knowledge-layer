package secondbrain

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatLinks struct {
	TelegramChatID   *string `json:"telegram_chat_id,omitempty"`
	MattermostUserID *string `json:"mattermost_user_id,omitempty"`
	UpdatedAt        string  `json:"updated_at"`
}

func GetChatLinks(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*ChatLinks, error) {
	var tg, mm *string
	var updated string
	err := pool.QueryRow(ctx, `
		SELECT telegram_chat_id, mattermost_user_id, updated_at::text
		FROM user_chat_links WHERE user_id=$1`, userID,
	).Scan(&tg, &mm, &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return &ChatLinks{UpdatedAt: ""}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("secondbrain: get chat links: %w", err)
	}
	return &ChatLinks{TelegramChatID: tg, MattermostUserID: mm, UpdatedAt: updated}, nil
}

func UpsertChatLinks(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, telegramChatID, mattermostUserID *string) (*ChatLinks, error) {
	cur, err := GetChatLinks(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	tg := cur.TelegramChatID
	mm := cur.MattermostUserID
	if telegramChatID != nil {
		tg = telegramChatID
	}
	if mattermostUserID != nil {
		mm = mattermostUserID
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO user_chat_links (user_id, telegram_chat_id, mattermost_user_id, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id) DO UPDATE SET
			telegram_chat_id = EXCLUDED.telegram_chat_id,
			mattermost_user_id = EXCLUDED.mattermost_user_id,
			updated_at = now()`, userID, tg, mm)
	if err != nil {
		return nil, fmt.Errorf("secondbrain: upsert chat links: %w", err)
	}
	return GetChatLinks(ctx, pool, userID)
}
