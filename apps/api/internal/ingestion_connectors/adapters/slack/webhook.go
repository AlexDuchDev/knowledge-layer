package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

// MaxWebhookSkew bounds how far a Slack request_timestamp may deviate from
// "now" before we treat the delivery as a replay attack. 5 minutes is the
// value Slack itself recommends in their verification doc.
const MaxWebhookSkew = 5 * time.Minute

// nowFn is overridable in tests so signature/replay assertions are deterministic.
var nowFn = time.Now

// Webhook errors are sentinel-style so the route layer can map to HTTP codes
// (400 for malformed body, 401 for bad signature / stale timestamp, 503 for
// missing signing secret) without parsing strings.
var (
	ErrWebhookSigningSecretMissing = errors.New("slack webhook: signing_secret not configured on feed")
	ErrWebhookBadSignature         = errors.New("slack webhook: signature mismatch")
	ErrWebhookStaleTimestamp       = errors.New("slack webhook: request timestamp outside acceptable skew")
	ErrWebhookMissingHeaders       = errors.New("slack webhook: missing X-Slack-Signature or X-Slack-Request-Timestamp")
	ErrWebhookMalformedBody        = errors.New("slack webhook: malformed JSON body")
)

// slackEnvelope is the outer Events API envelope shared by url_verification
// and event_callback deliveries. We decode lazily so verification can run on
// the raw bytes (HMAC requires byte-exact body) before we touch JSON.
type slackEnvelope struct {
	Type      string          `json:"type"`
	Challenge string          `json:"challenge,omitempty"`
	TeamID    string          `json:"team_id,omitempty"`
	APIAppID  string          `json:"api_app_id,omitempty"`
	EventID   string          `json:"event_id,omitempty"`
	EventTime int64           `json:"event_time,omitempty"`
	Event     json.RawMessage `json:"event,omitempty"`
}

// slackEventCore lifts only the fields we need for raw-artifact dedup keys
// and timestamps; the full event payload is preserved in Body verbatim.
type slackEventCore struct {
	Type      string `json:"type"`
	TS        string `json:"ts,omitempty"`
	EventTS   string `json:"event_ts,omitempty"`
	User      string `json:"user,omitempty"`
	Channel   string `json:"channel,omitempty"`
	ClientMsg string `json:"client_msg_id,omitempty"`
}

// HandleWebhook implements ingestion_connectors.WebhookHandler.
//
// Pipeline:
//  1. Pull X-Slack-Signature + X-Slack-Request-Timestamp.
//  2. Reject if either is missing or if timestamp is stale (replay protection).
//  3. Compute HMAC-SHA256("v0:" + ts + ":" + body, signing_secret) and
//     compare against the v0= prefix of the header (constant-time).
//  4. If body type=url_verification, return Challenge in WebhookResult.
//  5. If body type=event_callback, package the event as a single
//     "slack_event" RawArtifactInput and return.
//  6. All other types are accepted (signature was good) but produce no
//     artifacts — Slack sometimes sends event types we do not consume yet.
func (Adapter) HandleWebhook(ctx context.Context, in ingestion_connectors.WebhookRequest) (*ingestion_connectors.WebhookResult, error) {
	_ = ctx
	cfg, err := ingestion_connectors.ParseSlackFeedConfig(in.Feed.ConnectorConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("slack webhook: parse feed config: %w", err)
	}
	if strings.TrimSpace(cfg.SigningSecret) == "" {
		return nil, ErrWebhookSigningSecretMissing
	}

	sig := firstHeader(in.Headers, "X-Slack-Signature")
	tsHeader := firstHeader(in.Headers, "X-Slack-Request-Timestamp")
	if sig == "" || tsHeader == "" {
		return nil, ErrWebhookMissingHeaders
	}
	if err := verifySlackSignature(cfg.SigningSecret, tsHeader, in.Body, sig); err != nil {
		return nil, err
	}

	var env slackEnvelope
	if len(in.Body) > 0 {
		if err := json.Unmarshal(in.Body, &env); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrWebhookMalformedBody, err)
		}
	}

	switch env.Type {
	case "url_verification":
		return &ingestion_connectors.WebhookResult{Challenge: env.Challenge}, nil
	case "event_callback":
		art, err := buildArtifactFromEnvelope(env, in.Body)
		if err != nil {
			return nil, err
		}
		return &ingestion_connectors.WebhookResult{RawArtifacts: []ingestion_connectors.RawArtifactInput{art}}, nil
	default:
		// Unknown but signature-valid envelope (e.g. legacy outgoing webhook,
		// future event types). We accept silently rather than 4xx-ing Slack
		// into disabling the subscription.
		return &ingestion_connectors.WebhookResult{}, nil
	}
}

func firstHeader(h map[string][]string, key string) string {
	if h == nil {
		return ""
	}
	// Try canonical key first, then case-insensitive search to be tolerant of
	// callers that didn't run http.Header.Set.
	if v, ok := h[key]; ok && len(v) > 0 {
		return strings.TrimSpace(v[0])
	}
	for k, v := range h {
		if strings.EqualFold(k, key) && len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
	}
	return ""
}

func verifySlackSignature(signingSecret, timestamp string, body []byte, headerSig string) error {
	tsInt, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return fmt.Errorf("%w: timestamp not numeric: %v", ErrWebhookBadSignature, err)
	}
	now := nowFn()
	delta := now.Unix() - tsInt
	if delta < 0 {
		delta = -delta
	}
	if time.Duration(delta)*time.Second > MaxWebhookSkew {
		return ErrWebhookStaleTimestamp
	}

	// Slack base string: "v0:{timestamp}:{raw_body}"
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte("v0:" + strings.TrimSpace(timestamp) + ":"))
	mac.Write(body)
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Constant-time compare; lengths must match before subtle.ConstantTimeCompare.
	if len(expected) != len(headerSig) {
		return ErrWebhookBadSignature
	}
	if !hmac.Equal([]byte(expected), []byte(headerSig)) {
		return ErrWebhookBadSignature
	}
	return nil
}

// buildArtifactFromEnvelope packages the raw delivery body as a single
// "slack_event" raw artifact. We preserve the WHOLE envelope (not just .event)
// so future replay/normalisation has the team_id, api_app_id, event_id, and
// event_time without re-querying Slack.
func buildArtifactFromEnvelope(env slackEnvelope, rawBody []byte) (ingestion_connectors.RawArtifactInput, error) {
	var core slackEventCore
	if len(env.Event) > 0 {
		_ = json.Unmarshal(env.Event, &core) // best-effort; fields are optional
	}

	// Dedup key: prefer event_id (Slack guarantees uniqueness across a 30-day
	// retry window), then channel:ts (older events without event_id), then
	// fall back to a synthetic id from timestamp so DB unique-on-content still
	// fires if the same delivery hits twice.
	externalRef := env.EventID
	if externalRef == "" && core.Channel != "" && core.TS != "" {
		externalRef = core.Channel + ":" + core.TS
	}
	if externalRef == "" {
		externalRef = "slack_event:" + strconv.FormatInt(env.EventTime, 10)
	}

	var srcCreated *time.Time
	if env.EventTime > 0 {
		t := time.Unix(env.EventTime, 0).UTC()
		srcCreated = &t
	} else if core.EventTS != "" {
		if secs, err := slackTSToTime(core.EventTS); err == nil {
			srcCreated = &secs
		}
	} else if core.TS != "" {
		if secs, err := slackTSToTime(core.TS); err == nil {
			srcCreated = &secs
		}
	}

	return ingestion_connectors.RawArtifactInput{
		ArtifactType:    "slack_message",
		ExternalRef:     externalRef,
		Payload:         rawBody,
		SourceCreatedAt: srcCreated,
		SourceAuthorRef: core.User,
		ExtraMetadata: map[string]any{
			"slack_team_id":     env.TeamID,
			"slack_api_app_id":  env.APIAppID,
			"slack_event_id":    env.EventID,
			"slack_event_type":  core.Type,
			"slack_channel":     core.Channel,
			"slack_delivery_ts": env.EventTime,
		},
	}, nil
}

// slackTSToTime parses Slack's "1234567890.123456" string into time.Time (UTC).
func slackTSToTime(ts string) (time.Time, error) {
	parts := strings.SplitN(ts, ".", 2)
	secs, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	var nanos int64
	if len(parts) == 2 {
		// "123456" microseconds → ns.
		us, perr := strconv.ParseInt(parts[1], 10, 64)
		if perr == nil {
			nanos = us * 1000
		}
	}
	return time.Unix(secs, nanos).UTC(), nil
}
