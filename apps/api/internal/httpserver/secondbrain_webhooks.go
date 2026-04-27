package httpserver

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/qa"
	"github.com/knowledgelayer/api/internal/retrieval_intelligence"
	"github.com/knowledgelayer/api/internal/secondbrain"
)

func mountSecondBrainWebhooks(f *fiber.App, d *app.Deps, cfg config.Config) {
	sec := strings.TrimSpace(cfg.SecondBrainWebhookSecret)
	if sec == "" {
		return
	}
	base := "/webhooks/second-brain/" + sec
	f.Post(base+"/telegram", func(c *fiber.Ctx) error {
		return handleSecondBrainTelegram(c, d, cfg)
	})
	f.Post(base+"/mattermost", func(c *fiber.Ctx) error {
		return handleSecondBrainMattermost(c, d, cfg)
	})
}

func handleSecondBrainTelegram(c *fiber.Ctx, d *app.Deps, cfg config.Config) error {
	var raw map[string]any
	if err := json.Unmarshal(c.Body(), &raw); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}
	msg, _ := raw["message"].(map[string]any)
	if msg == nil {
		return c.SendStatus(fiber.StatusOK)
	}
	chat, _ := msg["chat"].(map[string]any)
	if chat == nil {
		return c.SendStatus(fiber.StatusOK)
	}
	chatStr := telegramChatString(chat["id"])
	text, _ := msg["text"].(string)
	text = strings.TrimSpace(text)
	if text == "" || chatStr == "" {
		return c.SendStatus(fiber.StatusOK)
	}

	var principal uuid.UUID
	err := d.Pool.QueryRow(c.Context(), `SELECT user_id FROM user_chat_links WHERE telegram_chat_id=$1`, chatStr).Scan(&principal)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.SendStatus(fiber.StatusOK)
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	reply, err := secondBrainBotAsk(c, d, cfg, principal, text, "telegram")
	if err != nil {
		reply = "Sorry, I could not answer: " + err.Error()
	}
	tok := strings.TrimSpace(cfg.TelegramBotToken)
	if tok != "" {
		_ = secondbrain.SendTelegramText(c.Context(), tok, chatStr, reply)
	}
	return c.SendStatus(fiber.StatusOK)
}

func telegramChatString(v any) string {
	switch t := v.(type) {
	case float64:
		return fmt.Sprintf("%.0f", t)
	case json.Number:
		return t.String()
	case string:
		return strings.TrimSpace(t)
	default:
		return ""
	}
}

func handleSecondBrainMattermost(c *fiber.Ctx, d *app.Deps, cfg config.Config) error {
	mt := strings.TrimSpace(cfg.MattermostOutgoingWebhookToken)
	if mt == "" {
		return fiber.NewError(fiber.StatusServiceUnavailable, "MATTERMOST_OUTGOING_WEBHOOK_TOKEN not configured")
	}
	if c.FormValue("token") != mt {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}
	mmUser := strings.TrimSpace(c.FormValue("user_id"))
	text := strings.TrimSpace(c.FormValue("text"))
	if mmUser == "" || text == "" {
		return c.JSON(map[string]string{"text": "missing user_id or text"})
	}
	var principal uuid.UUID
	err := d.Pool.QueryRow(c.Context(), `SELECT user_id FROM user_chat_links WHERE mattermost_user_id=$1`, mmUser).Scan(&principal)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.JSON(map[string]string{"text": "User is not linked in Knowledge Layer. Set mattermost_user_id via PUT /me/chat-links."})
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	reply, err := secondBrainBotAsk(c, d, cfg, principal, text, "mattermost")
	if err != nil {
		reply = "Sorry, I could not answer: " + err.Error()
	}
	return c.JSON(map[string]string{"text": reply})
}

func secondBrainBotAsk(c *fiber.Ctx, d *app.Deps, cfg config.Config, principal uuid.UUID, rawText, channel string) (string, error) {
	q := strings.TrimSpace(rawText)
	if strings.HasPrefix(strings.ToLower(q), "/ask") {
		q = strings.TrimSpace(q[4:])
	}
	if q == "" {
		return "Send a question after /ask.", nil
	}
	sc := strings.TrimSpace(cfg.SecondBrainBotScenarioCode)
	if sc != "" {
		ok, err := d.RoleBuilder.Assignments.PrincipalAllowsScenario(c.Context(), principal, sc)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("scenario not permitted for this user")
		}
	}
	askIn := qa.AskEntityInput{
		Question:       q,
		IncludeRelated: false,
		AnswerStrategy: "standard",
		ScenarioCode:   sc,
	}
	if err := d.Retrieval.PreprocessAskMultimodal(c.Context(), &askIn); err != nil {
		return "", err
	}
	filters := retrieval_intelligence.BuildGlobalAskSearchFilters(askIn.Question, "", "", "", "", "", "")
	out, _, err := d.Retrieval.AskGlobal(c.Context(), principal, askIn, filters, "")
	if err != nil {
		return "", err
	}
	_ = secondbrain.RecordProductEvent(c.Context(), d.Pool, "ask_command", &principal, nil, nil, map[string]any{"channel": channel})
	return out.Answer, nil
}
