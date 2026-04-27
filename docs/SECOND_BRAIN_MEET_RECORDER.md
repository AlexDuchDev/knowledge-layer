# Second Brain — свой сервер записи Google Meet

**Статус:** **вне объёма** текущего пилота ядра Knowledge Layer.

## Рекомендация

Использовать транскрипт-провайдера (например Fireflies), который уже интегрирован как коннектор, и политики из [FIREFLIES_SECURITY.md](./FIREFLIES_SECURITY.md).

## Если клиент настаивает на self-hosted записи

Выделить **отдельный сервис** (запись, хранение аудио, ASR, спикеры) с контрактом выхода: загрузка в тот же контур `raw_artifacts` / `fireflies_transcript` или новый `artifact_type` с нормализатором. Оценка: [SECOND_BRAIN_OVERLAY_SIZING.md](./SECOND_BRAIN_OVERLAY_SIZING.md) §3.
