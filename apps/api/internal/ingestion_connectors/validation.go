package ingestion_connectors

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Sync modes supported by the framework (declared on source_feeds.sync_mode).
const (
	SyncModeFullImport  = "full_import"
	SyncModeIncremental = "incremental"
	SyncModeEventDriven = "event_driven"
	SyncModeManual      = "manual"
)

var allowedSyncModes = map[string]struct{}{
	SyncModeFullImport:  {},
	SyncModeIncremental: {},
	SyncModeEventDriven: {},
	SyncModeManual:      {},
}

// ValidateSyncMode returns an error if the mode is not one of the framework values.
func ValidateSyncMode(mode string) error {
	if mode == "" {
		return nil
	}
	if _, ok := allowedSyncModes[mode]; !ok {
		return fmt.Errorf("sync_mode must be one of %s, %s, %s, %s", SyncModeFullImport, SyncModeIncremental, SyncModeEventDriven, SyncModeManual)
	}
	return nil
}

// ValidateCreateSourceFeedInput enforces governance-bearing fields before persistence.
func ValidateCreateSourceFeedInput(in CreateSourceFeedInput) error {
	if in.ConnectorID == uuid.Nil {
		return errors.New("connector_id required")
	}
	if strings.TrimSpace(in.DisplayName) == "" {
		return errors.New("display_name required")
	}
	if in.OwnerID == uuid.Nil {
		return errors.New("owner_id required")
	}
	if in.DomainID == uuid.Nil {
		return errors.New("domain_id required")
	}
	if err := ValidateSyncMode(in.SyncMode); err != nil {
		return err
	}
	var jobs []any
	if len(in.AllowedJobTypesJSON) == 0 {
		return errors.New("allowed_job_types_json required and must be non-empty")
	}
	if err := json.Unmarshal(in.AllowedJobTypesJSON, &jobs); err != nil {
		return fmt.Errorf("allowed_job_types_json: %w", err)
	}
	if len(jobs) == 0 {
		return errors.New("allowed_job_types_json must list at least one job type")
	}
	if in.IngestionMode == "" {
		return errors.New("ingestion_mode required")
	}
	if strings.TrimSpace(in.KnowledgeScope) == "" {
		return errors.New("knowledge_scope required")
	}
	return nil
}
