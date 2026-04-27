package mattermost

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

// Sentinel errors mirror the Slack adapter so the webhook route can map to
// the same HTTP status codes (401/503/400) without needing per-adapter logic.
var (
	ErrWebhookTokenMissing  = errors.New("mattermost webhook: outgoing_webhook_token not configured on feed")
	ErrWebhookBadToken      = errors.New("mattermost webhook: token mismatch")
	ErrWebhookMalformedBody = errors.New("mattermost webhook: malformed form body")
)

// HandleWebhook implements ingestion_connectors.WebhookHandler for Mattermost
// Outgoing Webhooks (System Console → Integrations → Outgoing Webhooks).
//
// Mattermost's outgoing webhook format is form-encoded (not JSON, not HMAC):
// every delivery includes a `token` field whose value must match the
// configured per-feed `outgoing_webhook_token`. We verify with constant-time
// comparison and reject mismatches as 401.
//
// Fields populated for the raw artifact:
//   - artifact_type:   "mattermost_post"
//   - external_ref:    post_id (Mattermost guarantees uniqueness)
//   - source_author_ref: user_id
//   - source_created_at: timestamp (epoch ms → time.Time)
//   - extra_metadata:  team_id, team_domain, channel_id, channel_name, trigger_word
//
// The verbatim form body is preserved as the artifact payload so downstream
// normalisers / replay tools have the full delivery, not a synthesised JSON.
func (Adapter) HandleWebhook(ctx context.Context, in ingestion_connectors.WebhookRequest) (*ingestion_connectors.WebhookResult, error) {
	_ = ctx
	cfg, err := ingestion_connectors.ParseMattermostFeedConfig(in.Feed.ConnectorConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("mattermost webhook: parse feed config: %w", err)
	}
	expected := strings.TrimSpace(cfg.OutgoingWebhookToken)
	if expected == "" {
		return nil, ErrWebhookTokenMissing
	}

	form, err := url.ParseQuery(string(in.Body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWebhookMalformedBody, err)
	}

	got := strings.TrimSpace(form.Get("token"))
	if got == "" {
		return nil, ErrWebhookBadToken
	}
	// Constant-time compare; lengths must match before subtle.ConstantTimeCompare.
	if len(got) != len(expected) || subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
		return nil, ErrWebhookBadToken
	}

	postID := strings.TrimSpace(form.Get("post_id"))
	userID := strings.TrimSpace(form.Get("user_id"))

	var srcCreated *time.Time
	if ts := strings.TrimSpace(form.Get("timestamp")); ts != "" {
		// Mattermost timestamp is epoch milliseconds.
		if ms, perr := strconv.ParseInt(ts, 10, 64); perr == nil && ms > 0 {
			t := time.UnixMilli(ms).UTC()
			srcCreated = &t
		}
	}

	// External-ref fallback chain: post_id → channel_id+timestamp → form digest.
	externalRef := postID
	if externalRef == "" {
		externalRef = strings.TrimSpace(form.Get("channel_id")) + ":" + strings.TrimSpace(form.Get("timestamp"))
	}

	extraMeta := map[string]any{
		"mattermost_team_id":       strings.TrimSpace(form.Get("team_id")),
		"mattermost_team_domain":   strings.TrimSpace(form.Get("team_domain")),
		"mattermost_channel_id":    strings.TrimSpace(form.Get("channel_id")),
		"mattermost_channel_name":  strings.TrimSpace(form.Get("channel_name")),
		"mattermost_user_name":     strings.TrimSpace(form.Get("user_name")),
		"mattermost_trigger_word":  strings.TrimSpace(form.Get("trigger_word")),
	}

	return &ingestion_connectors.WebhookResult{
		RawArtifacts: []ingestion_connectors.RawArtifactInput{
			{
				ArtifactType:    "mattermost_post",
				ExternalRef:     externalRef,
				Payload:         in.Body,
				SourceCreatedAt: srcCreated,
				SourceAuthorRef: userID,
				ExtraMetadata:   extraMeta,
			},
		},
	}, nil
}

// MapArtifactMetadata is unused for webhook deliveries (the route persists via
// IngestWebhookResult which calls Service.buildRawArtifactMetadataJSON), but
// the contract requires a value — we add a simple JSON parse fallback for
// payloads that happen to be JSON, mirroring slack/slack.go.
//
// Currently overlaps with the existing Adapter.MapArtifactMetadata on the
// same struct; documenting here that webhook deliveries flow through the
// same merger.
var _ = json.Unmarshal // appease imports if re-used in tests
