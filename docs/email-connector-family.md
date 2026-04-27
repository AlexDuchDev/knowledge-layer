# Email connector family

Shared **`NormalizedEmailMessage`** for Gmail and other mailbox connectors (`record_type = email_message`).

## Gmail v1

- `connector_config_json`: `gmail_oauth_access_token` (user OAuth access token).
- `external_ref`: optional Gmail search query passed to the messages list API (`q` parameter).

## Package

Go: `internal/ingestion_connectors/families/email`.

M365 Outlook mail uses the **microsoft365** family (`m365_mail_message`) via Microsoft Graph: same token as other Graph products, optional `mail_folder_id`, `shared_mailbox_upn`, and `mail_filter_query` in `connector_config_json`. Long-term, consider converging on `email_message` where semantics align.
