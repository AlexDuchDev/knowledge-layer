package ingestion_connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/chat"
)

type Connector struct {
	ID               uuid.UUID       `json:"id"`
	Type             string          `json:"type"`
	DisplayName      string          `json:"display_name"`
	AuthMode         string          `json:"auth_mode,omitempty"`
	Status           string          `json:"status"`
	AuthConfigRef    string          `json:"auth_config_ref,omitempty"`
	ConfigJSON       json.RawMessage `json:"config_json,omitempty"`
	CapabilitiesJSON json.RawMessage `json:"capabilities_json,omitempty"`
}

type SourceFeed struct {
	ID                  uuid.UUID       `json:"id"`
	ConnectorID         uuid.UUID       `json:"connector_id"`
	DisplayName         string          `json:"display_name"`
	OwnerID             uuid.UUID       `json:"owner_id"`
	OwnerTeamID         *uuid.UUID      `json:"owner_team_id,omitempty"`
	DomainID            uuid.UUID       `json:"domain_id"`
	SensitivityLevel    int             `json:"sensitivity_level"`
	AllowedJobTypesJSON json.RawMessage `json:"allowed_job_types_json"`
	IngestionMode       string          `json:"ingestion_mode"`
	SyncMode            string          `json:"sync_mode"`
	Status              string          `json:"status"`
	SyncStatus          string          `json:"sync_status"`
	ExternalRef         string          `json:"external_ref"`
	KnowledgeScope      string          `json:"knowledge_scope"`
	Notes               *string         `json:"notes,omitempty"`
	ConnectorConfigJSON json.RawMessage `json:"connector_config_json,omitempty"`
}

type Service struct {
	pool       *pgxpool.Pool
	HTTP       *http.Client
	Registry   *Registry
	gmailOAuth *oauth2.Config
	m365OAuth  *oauth2.Config
}

// ConfigureConnectorOAuth wires optional OAuth2 clients for Gmail / Microsoft Graph token refresh.
func (s *Service) ConfigureConnectorOAuth(gmail, m365 *oauth2.Config) {
	s.gmailOAuth = gmail
	s.m365OAuth = m365
}

func NewService(pool *pgxpool.Pool, registry *Registry) *Service {
	return &Service{pool: pool, HTTP: &http.Client{Timeout: 30 * time.Second}, Registry: registry}
}

func (s *Service) ListSourceFeeds(ctx context.Context) ([]SourceFeed, error) {
	rows, err := s.pool.Query(ctx, sourceFeedSelectFull+` ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SourceFeed
	for rows.Next() {
		var f SourceFeed
		if err := scanSourceFeedFull(rows, &f); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

// ListSourceFeedsInDomains returns source feeds limited to the given domain IDs,
// without exposing connector_config_json (it remains nil and is omitted in JSON).
func (s *Service) ListSourceFeedsInDomains(ctx context.Context, domainIDs []uuid.UUID, limit int) ([]SourceFeed, error) {
	if len(domainIDs) == 0 {
		return []SourceFeed{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, sourceFeedSelectPublic+`
		WHERE domain_id = ANY($1) AND status <> 'archived'
		ORDER BY created_at DESC
		LIMIT $2`, domainIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SourceFeed
	for rows.Next() {
		var f SourceFeed
		if err := scanSourceFeedPublic(rows, &f); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func (s *Service) GetSourceFeed(ctx context.Context, id uuid.UUID) (*SourceFeed, error) {
	var f SourceFeed
	err := scanSourceFeedFull(s.pool.QueryRow(ctx, sourceFeedSelectFull+` WHERE id=$1`, id), &f)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

type CreateSourceFeedInput struct {
	ConnectorID         uuid.UUID       `json:"connector_id"`
	DisplayName         string          `json:"display_name"`
	OwnerID             uuid.UUID       `json:"owner_id"`
	OwnerTeamID         *uuid.UUID      `json:"owner_team_id,omitempty"`
	DomainID            uuid.UUID       `json:"domain_id"`
	SensitivityLevel    int             `json:"sensitivity_level"`
	AllowedJobTypesJSON json.RawMessage `json:"allowed_job_types_json"`
	IngestionMode       string          `json:"ingestion_mode"`
	SyncMode            string          `json:"sync_mode"`
	ExternalRef         string          `json:"external_ref"`
	KnowledgeScope      string          `json:"knowledge_scope"`
	Notes               *string         `json:"notes,omitempty"`
	ConnectorConfigJSON json.RawMessage `json:"connector_config_json"`
}

// CandidateSourceFeed builds the in-memory SourceFeed shape adapter validators
// see at config-save time (POST /source-feeds and POST /source-feeds/validate).
// Adapters such as filesystem read fields like ExternalRef directly, so the
// candidate must mirror **every** input field that an adapter could legitimately
// inspect. Earlier route-level constructions silently dropped ExternalRef, which
// made the filesystem and similar no-credentials connectors unusable from the UI.
// Centralising the conversion here prevents that drift class.
func (in CreateSourceFeedInput) CandidateSourceFeed() *SourceFeed {
	return &SourceFeed{
		ConnectorID:         in.ConnectorID,
		DisplayName:         in.DisplayName,
		OwnerID:             in.OwnerID,
		OwnerTeamID:         in.OwnerTeamID,
		DomainID:            in.DomainID,
		SensitivityLevel:    in.SensitivityLevel,
		AllowedJobTypesJSON: in.AllowedJobTypesJSON,
		IngestionMode:       in.IngestionMode,
		SyncMode:            in.SyncMode,
		ExternalRef:         in.ExternalRef,
		KnowledgeScope:      in.KnowledgeScope,
		Notes:               in.Notes,
		ConnectorConfigJSON: in.ConnectorConfigJSON,
	}
}

func (s *Service) CreateSourceFeed(ctx context.Context, in CreateSourceFeedInput) (*SourceFeed, error) {
	if len(in.AllowedJobTypesJSON) == 0 {
		in.AllowedJobTypesJSON = []byte("[]")
	}
	if len(in.ConnectorConfigJSON) == 0 {
		in.ConnectorConfigJSON = []byte("{}")
	}
	if in.IngestionMode == "" {
		in.IngestionMode = "ingestion_only"
	}
	if in.SyncMode == "" {
		in.SyncMode = SyncModeManual
	}
	if in.KnowledgeScope == "" {
		in.KnowledgeScope = "domain_linked"
	}
	if err := ValidateCreateSourceFeedInput(in); err != nil {
		return nil, err
	}
	id := uuid.New()
	sourceURI := in.ExternalRef
	_, err := s.pool.Exec(ctx, `
		INSERT INTO source_feeds (id, connector_id, source_uri, external_ref, display_name, owner_id, owner_team_id, domain_id, sensitivity_level,
			allowed_job_types_json, ingestion_mode, sync_mode, knowledge_scope, notes, connector_config_json, status, sync_status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,'draft','idle')`,
		id, in.ConnectorID, sourceURI, in.ExternalRef, in.DisplayName, in.OwnerID, in.OwnerTeamID, in.DomainID, in.SensitivityLevel,
		in.AllowedJobTypesJSON, in.IngestionMode, in.SyncMode, in.KnowledgeScope, in.Notes, in.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	return s.GetSourceFeed(ctx, id)
}

func (s *Service) PatchSourceFeed(ctx context.Context, id uuid.UUID, displayName *string, config json.RawMessage) (*SourceFeed, error) {
	if displayName != nil {
		_, err := s.pool.Exec(ctx, `UPDATE source_feeds SET display_name=$2, updated_at=now() WHERE id=$1`, id, *displayName)
		if err != nil {
			return nil, err
		}
	}
	if len(config) > 0 {
		_, err := s.pool.Exec(ctx, `UPDATE source_feeds SET connector_config_json=$2, updated_at=now() WHERE id=$1`, id, config)
		if err != nil {
			return nil, err
		}
	}
	return s.GetSourceFeed(ctx, id)
}

// ValidateSourceFeed runs adapter-level validation against a candidate feed
// (Phase 4.2.2). Returns nil when the adapter accepts the config or when no
// adapter is registered for the connector type. The route layer should call
// this on POST /source-feeds and PATCH /source-feeds/:id so that bad configs
// are rejected at save time rather than surfacing at sync execution.
//
// Distinct from CanActivate: CanActivate also enforces governance fields
// (owner, domain, knowledge_scope, allowed_job_types_json, ingestion_mode);
// ValidateSourceFeed is just the adapter contract. Both run together at
// activation; only this one runs at config-save.
func (s *Service) ValidateSourceFeed(ctx context.Context, feed *SourceFeed) error {
	if feed == nil {
		return errors.New("validate: nil feed")
	}
	if s.Registry == nil || s.pool == nil {
		return nil
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return fmt.Errorf("validate: load connector: %w", err)
	}
	a, err := s.Registry.AdapterForConnectorType(conn.Type)
	if err != nil {
		// Unknown connector type — surface as a validation error rather than
		// silently passing. A feed referencing an unregistered type would
		// fail at sync time anyway.
		return fmt.Errorf("validate: %w", err)
	}
	if err := a.ValidateConnectorConfig(ctx, *conn); err != nil {
		return fmt.Errorf("connector: %w", err)
	}
	if err := a.ValidateSourceFeedConfig(ctx, *conn, feed); err != nil {
		return fmt.Errorf("source_feed: %w", err)
	}
	return nil
}

func (s *Service) setFeedStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := s.pool.Exec(ctx, `UPDATE source_feeds SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}

func (s *Service) CanActivate(ctx context.Context, f *SourceFeed) error {
	if f.OwnerID == uuid.Nil {
		return errors.New("owner required")
	}
	if f.DomainID == uuid.Nil {
		return errors.New("domain required")
	}
	if err := ValidateSyncMode(f.SyncMode); err != nil {
		return err
	}
	if f.KnowledgeScope == "" {
		return errors.New("knowledge_scope required")
	}
	var jobs []any
	if err := json.Unmarshal(f.AllowedJobTypesJSON, &jobs); err != nil {
		return fmt.Errorf("allowed_job_types_json: %w", err)
	}
	if len(jobs) == 0 {
		return errors.New("allowed_job_types_json must be non-empty")
	}
	if f.IngestionMode == "" {
		return errors.New("ingestion_mode required")
	}
	if s.Registry != nil && s.pool != nil {
		conn, err := s.GetConnector(ctx, f.ConnectorID)
		if err != nil {
			return err
		}
		if a, err := s.Registry.AdapterForConnectorType(conn.Type); err == nil {
			if err := a.ValidateSourceFeedConfig(ctx, *conn, f); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) Activate(ctx context.Context, id uuid.UUID) error {
	f, err := s.GetSourceFeed(ctx, id)
	if err != nil {
		return err
	}
	if err := s.CanActivate(ctx, f); err != nil {
		return err
	}
	return s.setFeedStatus(ctx, id, "active")
}

func (s *Service) Pause(ctx context.Context, id uuid.UUID) error {
	return s.setFeedStatus(ctx, id, "paused")
}

func (s *Service) Resume(ctx context.Context, id uuid.UUID) error {
	return s.setFeedStatus(ctx, id, "active")
}

// ArchiveSourceFeed sets status to archived (soft delete).
func (s *Service) ArchiveSourceFeed(ctx context.Context, id uuid.UUID) error {
	return s.setFeedStatus(ctx, id, "archived")
}

// SyncTelegram performs manual Telegram sync: raw artifacts + normalized records only.
func (s *Service) SyncTelegram(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "telegram" {
		return nil, fmt.Errorf("connector is %s, not telegram", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}

	tgCfg, err := ParseTelegramFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := ValidateTelegramV1ForActivation(feed, tgCfg); err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	primaryChatID, err := ParseTelegramExternalChatID(feed.ExternalRef)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, true)
		return nil, err
	}

	offset := TelegramNextGetUpdatesOffset(tgCfg)
	allUpdates, warnings, fetchErrs := s.fetchTelegramUpdates(tgCfg.BotToken, offset)
	allowed := allowedChatSet(tgCfg.AllowedChatIDs)
	updates := filterTelegramUpdatesForFeed(allUpdates, primaryChatID, allowed)
	maxAck := maxTelegramUpdateID(allUpdates)

	ingested := 0
	deduped := 0
	errs := fetchErrs

	seen := map[string]bool{}
	for _, u := range updates {
		payload, err := json.Marshal(u)
		if err != nil {
			errs++
			continue
		}
		h := hashBytes(payload)
		if seen[h] {
			deduped++
			continue
		}
		seen[h] = true

		extID := fmt.Sprintf("tg-%v", u.UpdateID)
		var updateObj map[string]any
		if err := json.Unmarshal(payload, &updateObj); err != nil {
			errs++
			continue
		}
		extra := map[string]any{"telegram_update": updateObj}
		metaJSON, err := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, "telegram_update", payload, extra)
		if err != nil {
			errs++
			continue
		}
		rawID, inserted, err := insertRawArtifactRow(ctx, s.pool, feedID, runID, "telegram_update", extID, h, "", metaJSON, nil, nil)
		if err != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}

		normMsg := chat.FromTelegramUpdate(feedID, u.UpdateID, u.Message.MessageID, u.Message.Date, u.Message.Text, u.Message.Chat.ID)
		normPayload, err := json.Marshal(normMsg)
		if err != nil {
			errs++
			continue
		}
		nh := hashBytes(normPayload)
		tag, err := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, chat.RecordTypeChatMessage, normPayload, nh)
		if err != nil {
			errs++
			continue
		}
		if tag.RowsAffected() == 0 {
			deduped++
			continue
		}
		ingested++
	}

	status := syncRunStatusFromCounts(ingested, errs)
	s.finalizeIngestionRun(ctx, runID, status, ingested, deduped, warnings, errs)
	s.completeSourceFeedSync(ctx, feedID, errs, true)

	if maxAck > 0 {
		newCfg, err := MergeTelegramLastUpdateID(feed.ConnectorConfigJSON, maxAck)
		if err == nil {
			_, _ = s.pool.Exec(ctx, `UPDATE source_feeds SET connector_config_json=$2, updated_at=now() WHERE id=$1`, feedID, newCfg)
		}
	}

	return s.GetIngestionRun(ctx, runID)
}

func (s *Service) fetchTelegramUpdates(botToken string, offset int64) ([]tgUpdate, int, int) {
	u := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=1", botToken)
	if offset > 0 {
		u = fmt.Sprintf("%s&offset=%d", u, offset)
	}
	resp, err := s.HTTP.Get(u)
	if err != nil {
		return nil, 0, 1
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, 1
	}
	var parsed tgGetUpdatesResp
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
		return nil, 0, 1
	}
	if !parsed.OK {
		return nil, 1, 1
	}
	return parsed.Result, 0, 0
}

func extractMessage(u tgUpdate) map[string]any {
	m := map[string]any{"update_id": u.UpdateID}
	if u.Message != nil {
		m["message_id"] = u.Message.MessageID
		m["date"] = u.Message.Date
		m["text"] = u.Message.Text
		m["chat_id"] = u.Message.Chat.ID
	}
	return m
}

func hashBytes(b []byte) string {
	// short stable hash prefix for dedup key (not cryptographic requirement)
	h := uuid.NewSHA1(uuid.NameSpaceOID, b)
	return h.String()
}
