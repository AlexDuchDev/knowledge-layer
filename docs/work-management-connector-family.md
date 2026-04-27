# Work management connector family

Jira, Trello, and similar systems share **`NormalizedWorkItem`** (`record_type = work_item`) for issues/cards.

## Record types

| `record_type` | Meaning |
|---------------|---------|
| `work_item` | Issue or card with title, body text, status, assignee hint |

## Raw artifacts

- `work_mgmt_issue` (generic), plus vendor types `jira_issue`, `trello_card`, `asana_task`, `asana_story`, `linear_issue`, `linear_comment`

## Jira v1

- `connector_config_json`: `jira_site_base_url`, `jira_email`, `jira_api_token` (HTTP Basic to Jira Cloud REST); optional `jira_max_results` (issue search page size, default 50, max 100).
- `external_ref`: project key (JQL `project=KEY`).
- Onboarding: `POST /integrations/jira/list-projects` lists projects the token can see so admins pick a project instead of typing the key.

## Trello v1

- `connector_config_json`: `trello_api_key`, `trello_token`.
- `external_ref`: board id.
- Onboarding: `POST /integrations/trello/list-boards`.

## Asana v1

- `connector_config_json`: `asana_personal_access_token`.
- `external_ref`: project gid; tasks (cap 25) and stories per task ingested as separate raw artifacts.
- Onboarding: `POST /integrations/asana/list-projects`.

## Linear v1

- `connector_config_json`: `linear_api_key` (GraphQL `Authorization` header).
- `external_ref`: team id; issues (cap 25) with bounded comments per issue.
- Onboarding: `POST /integrations/linear/list-teams`.

## Package

Go: `internal/ingestion_connectors/families/work_mgmt`.
