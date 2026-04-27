# ADR-0012: Second Brain pilot scope and defaults

## Status

**Accepted (2026-04-24).** Promoted from Provisional during Phase 1 alignment so the Second Brain overlay can ship as an **optional, feature-flag-gated module** in OSS v1 without blocking on the original client SOW. Future scope changes follow the standard ADR process.

## OSS inclusion

Second Brain is **shipped in the open-source v1 release as an optional module**:

- **Default state:** disabled. Without the flags below the `internal/secondbrain/` queue handlers and `/meeting-tasks` UI surface produce no work and no records.
- **Enable flags:** `SECOND_BRAIN_PREBRIEF_TICK=1` (jobworker pre-brief polling) plus `TELEGRAM_BOT_TOKEN` and/or `MATTERMOST_OUTGOING_WEBHOOK_TOKEN` for outbound channels. See [CONFIG_ENV.md](../CONFIG_ENV.md) for the full env surface.
- **Why optional:** Second Brain has external dependencies (Telegram bot, Fireflies API, calendar polling) that are operationally heavy for a default deployment. Marking it optional keeps the OSS core focused while preserving the existing investment.
- **Future direction:** Phase 4 of the alignment plan may extract `internal/secondbrain/` into a plugin contract (extension point), at which point this ADR will be superseded. Until then, Second Brain stays inside the monorepo as an optional module gated by env.
- **Disclosure:** [OSS_V1_SCOPE.md](../OSS_V1_SCOPE.md) lists Second Brain under "Optional modules" and [LIMITATIONS.md](../LIMITATIONS.md) describes its no-op behavior when flags are not set.

## Context

Пилот Second Brain поверх KL требует решений по триггерам брифов, каналам доставки, источнику транскриптов и витрине метрик ([SECOND_BRAIN_OVERLAY_SIZING.md](../SECOND_BRAIN_OVERLAY_SIZING.md) §8). Команда разработки использует **нижеперечисленные допущения** как defaults для пилота; они уточняются в follow-up ADR при появлении конкретных требований клиента.

## Decision (pilot defaults)

1. **Pre-meeting триггер:** периодический **polling** (воркер раз в минуту при включённом флаге окружения), а не Google Calendar push в первой интеграции. Push может заменить polling в отдельном ADR без ломки схемы `pre_meeting_brief_queue`.
2. **Таймзона окна брифа:** время события и окно «за 10 минут» в **UTC** на первом шаге; привязка к профилю пользователя — follow-up.
3. **Канал пилота доставки:** сначала **REST + веб**; Telegram/Mattermost — через таблицу `user_chat_links` и последующее включение исходящих сообщений (см. [SECOND_BRAIN_BOTS.md](../SECOND_BRAIN_BOTS.md) при появлении).
4. **Транскрипты:** **Fireflies** (или уже подключённый провайдер) как источник; **свой** Meet recorder + ASR **не** входит в пилотный объём кода ядра ([SECOND_BRAIN_MEET_RECORDER.md](../SECOND_BRAIN_MEET_RECORDER.md)).
5. **OKR-витрина:** события в таблице `second_brain_product_events` + **SQL/BI** как основная витрина; in-app дашборды — отдельная фаза по запросу.

## Consequences

- Можно реализовывать API извлечённых задач, очередь pre-brief и события без обратной связи от клиента.
- При смене допущений обновить этот ADR и [SECOND_BRAIN_SCOPE_DECISIONS.md](../SECOND_BRAIN_SCOPE_DECISIONS.md).

## Related

- [SECOND_BRAIN_OVERLAY_SIZING.md](../SECOND_BRAIN_OVERLAY_SIZING.md) §8
- [SECOND_BRAIN_SCOPE_DECISIONS.md](../SECOND_BRAIN_SCOPE_DECISIONS.md)
- [EXTRACTED_MEETING_TASKS.md](../EXTRACTED_MEETING_TASKS.md)
