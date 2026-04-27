---
name: Bug report
about: Something behaves wrong, crashes, or violates the contract in docs/.
title: "[bug] "
labels: [bug, triage]
---

<!--
Thanks for filing a bug. Before submitting:
- Search existing issues — yours might already be open or fixed.
- For SECURITY issues, do NOT use this template — see SECURITY.md.
- For docs gaps, use the "Documentation improvement" template instead.
-->

## What happened

<!-- Concrete, single-paragraph description. What did you do, what did you see, what did you expect? -->

## Reproduction

Minimal steps that produce the bug. Trim ruthlessly.

```bash
# Example
make db-up
cd apps/api && go run ./cmd/api
curl -X POST http://localhost:8080/ask -d '{"question":"…"}'
```

Expected: <!-- … -->
Actual: <!-- … -->

## Environment

- **Knowledge Layer version / commit:** <!-- `git rev-parse --short HEAD` or release tag -->
- **APP_ENV:** local / staging / production
- **OS + arch:** macOS arm64 / Linux x86_64 / …
- **Postgres version:** 16.x
- **Browser** (if a UI bug): Chrome 1XX / Firefox 1XX / …
- **Optional modules enabled:** Neo4j? Second Brain? OpenAI? — anything set via env that diverges from the defaults.

## Logs / output

<details>
<summary>API logs (last ~50 lines)</summary>

```text
<!-- paste here -->
```

</details>

<details>
<summary>Worker logs (if relevant)</summary>

```text
<!-- jobworker / connectorworker output -->
```

</details>

## Audit / observability checks already done

- [ ] Checked `GET /audit-events?event_type=…` for related entries.
- [ ] Checked `/ops/failed-runs` for failed ingestion or job runs.
- [ ] Checked `/ops/health` (worker `/ops/health` if a worker is involved).
- [ ] Reproduced on a clean DB (`make db-down && make db-up`).

## Anything else

<!-- Workarounds you tried, related issues, screenshots, hunches. -->
