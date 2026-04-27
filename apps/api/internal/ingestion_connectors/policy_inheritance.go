package ingestion_connectors

import (
	"encoding/json"

	"github.com/google/uuid"
)

// MergeFeedPolicyMetadata embeds governance defaults from the source feed into artifact metadata.
// Downstream normalization and entity creation should treat these as inherited policy, not optional hints.
func MergeFeedPolicyMetadata(feed SourceFeed, base map[string]any) map[string]any {
	if base == nil {
		base = make(map[string]any)
	}
	gov := map[string]any{
		"domain_id":         feed.DomainID.String(),
		"sensitivity_level": feed.SensitivityLevel,
		"source_feed_id":    feed.ID.String(),
		"owner_id":          feed.OwnerID.String(),
		"connector_id":      feed.ConnectorID.String(),
		"ingestion_mode":    feed.IngestionMode,
		"sync_mode":         feed.SyncMode,
		"knowledge_scope":   feed.KnowledgeScope,
		"external_ref":      feed.ExternalRef,
	}
	if feed.OwnerTeamID != nil {
		gov["owner_team_id"] = feed.OwnerTeamID.String()
	}
	if len(feed.AllowedJobTypesJSON) > 0 {
		var jobs any
		if err := json.Unmarshal(feed.AllowedJobTypesJSON, &jobs); err == nil {
			gov["allowed_job_types"] = jobs
		}
	}
	base["governance"] = gov
	return base
}

// FeedPolicySnapshot is a typed view of inherited governance for tests and workers.
type FeedPolicySnapshot struct {
	DomainID         uuid.UUID  `json:"domain_id"`
	SensitivityLevel int        `json:"sensitivity_level"`
	SourceFeedID     uuid.UUID  `json:"source_feed_id"`
	OwnerID          uuid.UUID  `json:"owner_id"`
	OwnerTeamID      *uuid.UUID `json:"owner_team_id,omitempty"`
	KnowledgeScope   string     `json:"knowledge_scope"`
	ExternalRef      string     `json:"external_ref"`
	IngestionMode    string     `json:"ingestion_mode"`
	SyncMode         string     `json:"sync_mode"`
	AllowedJobsJSON  []byte     `json:"-"`
}

// PolicySnapshotFromFeed builds a snapshot from a loaded feed row.
func PolicySnapshotFromFeed(f SourceFeed) FeedPolicySnapshot {
	s := FeedPolicySnapshot{
		DomainID:         f.DomainID,
		SensitivityLevel: f.SensitivityLevel,
		SourceFeedID:     f.ID,
		OwnerID:          f.OwnerID,
		OwnerTeamID:      f.OwnerTeamID,
		KnowledgeScope:   f.KnowledgeScope,
		ExternalRef:      f.ExternalRef,
		IngestionMode:    f.IngestionMode,
		SyncMode:         f.SyncMode,
		AllowedJobsJSON:  append([]byte(nil), f.AllowedJobTypesJSON...),
	}
	return s
}
