# Second Brain — исходящие боты (Telegram / Mattermost)

**Статус:** v1 — маппинг пользователя, webhooks с секретом в URL, опциональная очередь Asynq для исходящих; Mattermost исходящий POST в канал API пока не реализован (входящий webhook отвечает JSON `text`).

## Что уже есть

- **Ingestion:** коннекторы Telegram и Mattermost пишут в `raw_artifacts` / нормализацию ([CONNECTOR_CAPABILITY_MATRIX.md](./CONNECTOR_CAPABILITY_MATRIX.md)).

## Что добавлено в коде (v1)

- Таблица **`user_chat_links`** (миграция `000038`): привязка пользователя KL к внешнему идентификатору для доставки (`telegram_chat_id`, `mattermost_user_id`).
- HTTP (сессия / dev header): **`GET /me/chat-links`**, **`PUT /me/chat-links`**.
- **Webhooks** (без сессии; включены только если задан `SECOND_BRAIN_WEBHOOK_SECRET`):
  - `POST /webhooks/second-brain/<SECRET>/telegram` — тело: Telegram Bot API `Update` (JSON). Сообщение `text` может начинаться с `/ask`. Ответ в чат — через `TELEGRAM_BOT_TOKEN` (если задан). Логика вопроса: тот же retrieval stack, что и `POST /ask` ([`secondbrain_webhooks.go`](../apps/api/internal/httpserver/secondbrain_webhooks.go)).
  - `POST /webhooks/second-brain/<SECRET>/mattermost` — `application/x-www-form-urlencoded`, поля `user_id`, `text`; при установленном `MATTERMOST_OUTGOING_WEBHOOK_TOKEN` обязателен form-field `token` с тем же значением. Ответ: `{"text":"..."}` для outgoing webhook.
- **Очередь:** тип Asynq `secondbrain:outbound` (`cmd/jobworker`) — доставка текста в Telegram по связке `user_chat_links` + `TELEGRAM_BOT_TOKEN`; канал `mattermost` в воркере пока no-op.
- **События продукта:** `ask_command`, `brief_sent` и др. пишутся в `second_brain_product_events` (см. [API_SURFACE_V1.md](./API_SURFACE_V1.md) §14.1).

Переменные окружения: [CONFIG_ENV.md](./CONFIG_ENV.md) (раздел Second Brain).

## Следующие шаги

- Верификация владения Telegram-чатом (deep-link / код).
- Mattermost: исходящие сообщения через Bot API + site URL.
- Rate limits на webhooks и отдельный лимитер для `/ask` из ботов.

См. [SECOND_BRAIN_OVERLAY_SIZING.md](./SECOND_BRAIN_OVERLAY_SIZING.md) §1.
