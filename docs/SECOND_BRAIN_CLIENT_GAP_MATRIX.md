# Second Brain (client brief) ↔ Knowledge Layer — capability matrix

Purpose: align conversations with a client who arrived with an “AI Second Brain” MVP spec. This is **not** a spec-to-code contract: KL product decisions stay with us; the matrix shows **where KL already covers the intent** and **what we chose to build, defer, or skip** (aligned with the agreed internal backlog, not the client’s sprint plan).

Legend: **Yes** = supported today · **Partial** = same direction, different shape · **Planned** = agreed backlog · **Deferred** = consciously later · **N/A** = out of scope for KL core


| Client theme (from their doc)       | KL module / surface                                                 | Status   | Notes                                                                                                                |
| ----------------------------------- | ------------------------------------------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------- |
| Г1.1 Unified comms layer            | Connectors, `raw_artifacts`, `normalized_records`, connector worker | Partial  | Same architecture; channel set differs per deployment                                                                |
| Г1.1 Google Meet transcripts        | `fireflies` + `meeting_transcript` + optional calendar context      | Yes      | Via Fireflies (not self-hosted Meet recorder)                                                                        |
| Г1.1 Telegram chats                 | `telegram` connector, `chat` normalization                          | Yes      | Ingestion-first; delivery bots are a separate product decision                                                       |
| Г1.1 Mattermost                     | `mattermost` connector (v1 channel posts)                           | Yes      | Ingestion + `mattermost_post` normalization; delivery bots still separate — see [SECOND_BRAIN_OVERLAY_SIZING.md](./SECOND_BRAIN_OVERLAY_SIZING.md) |
| Г1.2 Knowledge graph                | `entities`, `entity_links`, domains, explorer                       | Partial  | Relational + graph-like links, not Neo4j                                                                             |
| Г1.2 Calendar title `Project.Topic` | `calendar_event` payload enrichment                                 | Planned  | Parser + optional link hints; see [meeting-transcript-connector-family.md](./meeting-transcript-connector-family.md) |
| Г2.1 Task extraction                | `extracted_meeting_tasks` + review events                           | Planned  | Schema + lifecycle doc: [EXTRACTED_MEETING_TASKS.md](./EXTRACTED_MEETING_TASKS.md)                                   |
| Г2.2 Decision extraction            | Knowledge job `decision_extraction`                                 | Yes      | Job-driven, not ad-hoc REST                                                                                          |
| Г4.1 Pre-meeting brief (10 min)     | Calendar-driven notifications                                       | Deferred | Requires event/window triggers beyond current scheduled tick                                                         |
| Г5.1 Project assistant              | `POST /ask`, citations, traces, domain scope                        | Yes      | Web-first; TG/MM assistants not implied                                                                              |
| Г3.1 Jira                           | `jira` connector → `work_item`                                      | Yes      | Outside their MVP but available in KL                                                                                |
| Row-level security in Postgres      | App-layer `AccessEvaluator` + SQL scoping                           | Partial  | Argue equivalence with security reviewers                                                                            |
| LLM eval / OSS fine-tune            | —                                                                   | N/A      | Research track, not platform core                                                                                    |


## Pre-stage items from the client (speaker accuracy, model shootouts)


| Topic                             | KL stance                                                                                                  |
| --------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| Diarization + email mapping       | Rely on transcript provider (Fireflies) + governance; see [FIREFLIES_SECURITY.md](./FIREFLIES_SECURITY.md) |
| Benchmark Gemini vs Claude vs GPT | Client-owned or consulting; not shipped in OSS core                                                        |


## How to use this with the client

1. Show **Yes** rows first — prove the platform already matches the spine of their story.
2. Use **Planned** rows to sequence work (tasks table, calendar parsing, extraction UI/API).
3. Use **Deferred / N/A** to reset expectations without blocking a pilot on KL.
4. For **effort ranges and pre-kickoff questions** (bots, briefs, Meet source, OKR depth), use [SECOND_BRAIN_OVERLAY_SIZING.md](./SECOND_BRAIN_OVERLAY_SIZING.md) §8.

