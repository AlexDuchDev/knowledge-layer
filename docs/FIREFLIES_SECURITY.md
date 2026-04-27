# Fireflies.ai connector — security and data handling

Fireflies is a **third-party** that holds meeting audio/transcripts and exposes them via API. Using it inside Knowledge Layer shifts trust boundaries: KL stores normalized copies and metadata; operators must treat tokens and transcript bodies as **high sensitivity**.

## Threat model (practical)


| Risk                                     | Mitigation                                                                                                                                                                            |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| API key leakage                          | Store only in `source_feeds.connector_config_json` (encrypted at rest if your deployment encrypts DB); restrict who can `GET /source-feeds/:id` with config; rotate keys on schedule. |
| Over-broad ingestion                     | One **source feed** per calendar/workspace boundary; minimum domain sensitivity; narrow allowed job types.                                                                            |
| Unauthorized transcript visibility in KL | Same as any document: domain grants, entity ACLs, `view_raw` / sensitivity policies; prefer publishing **extracted tasks/decisions** over exposing full transcript text to all users. |
| Data residency / subprocessors           | Complete customer DPIA using Fireflies’ DPA and subprocessors list; document in your org’s records of processing.                                                                     |
| Retention                                | Align KL retention (and optional blob store) with legal hold; delete raw artifacts when policy requires — KL supports reprocessing from raw where retained.                           |


## Operational checklist

1. **Identity**: service account / API key owned by a named role, not shared developer keys.
2. **Network**: TLS-only calls to Fireflies; no logging of bearer tokens.
3. **Principle of least privilege**: Fireflies API key scoped to minimum needed scopes (read transcripts only).
4. **Onboarding**: train users that meeting bots imply consent where legally required.
5. **Incident response**: procedure if Fireflies or key is compromised — revoke key, audit `raw_artifacts` for exposure window.

## Product stance

We **do not** replicate Fireflies’ diarization stack in KL v1. Accuracy and speaker labels are **provider responsibilities**; KL focuses on governed copy, extraction, and access control. If speaker quality is insufficient, the remediation path is **provider change / contract** or a **separate ASR project**, not a silent fork inside the connector.

## Related docs

- [meeting-transcript-connector-family.md](./meeting-transcript-connector-family.md)  
- [PRODUCTION_HARDENING.md](./PRODUCTION_HARDENING.md)  
- [ACCESS_MODEL.md](./ACCESS_MODEL.md)

