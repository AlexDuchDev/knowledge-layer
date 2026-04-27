package secondbrain

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const telegramMaxMessage = 3900

// SendTelegramText posts a plain-text message to a chat using the Bot API.
func SendTelegramText(ctx context.Context, botToken, chatID, text string) error {
	if strings.TrimSpace(botToken) == "" || strings.TrimSpace(chatID) == "" {
		return nil
	}
	body := strings.TrimSpace(text)
	if len(body) > telegramMaxMessage {
		body = body[:telegramMaxMessage] + "…"
	}
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.telegram.org/bot"+strings.TrimSpace(botToken)+"/sendMessage",
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("secondbrain: telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("secondbrain: telegram post: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("secondbrain: telegram status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// SendTelegramToUser loads telegram_chat_id for the user and sends text.
func SendTelegramToUser(ctx context.Context, pool *pgxpool.Pool, botToken string, userID uuid.UUID, text string) error {
	var chatID *string
	err := pool.QueryRow(ctx, `SELECT telegram_chat_id FROM user_chat_links WHERE user_id=$1`, userID).Scan(&chatID)
	if err != nil || chatID == nil || strings.TrimSpace(*chatID) == "" {
		return nil
	}
	return SendTelegramText(ctx, botToken, strings.TrimSpace(*chatID), text)
}
