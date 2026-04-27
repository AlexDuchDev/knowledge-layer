package ingestion_connectors

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const sourceFeedSelectFull = `
		SELECT id, connector_id, display_name, owner_id, owner_team_id, domain_id, sensitivity_level,
		       allowed_job_types_json, ingestion_mode, sync_mode, status, sync_status,
		       external_ref, knowledge_scope, notes, connector_config_json
		FROM source_feeds`

const sourceFeedSelectPublic = `
		SELECT id, connector_id, display_name, owner_id, owner_team_id, domain_id, sensitivity_level,
		       allowed_job_types_json, ingestion_mode, sync_mode, status, sync_status,
		       external_ref, knowledge_scope, notes
		FROM source_feeds`

func scanSourceFeedFull(row pgx.Row, f *SourceFeed) error {
	var ownerTeam pgtype.UUID
	var notes pgtype.Text
	err := row.Scan(&f.ID, &f.ConnectorID, &f.DisplayName, &f.OwnerID, &ownerTeam, &f.DomainID,
		&f.SensitivityLevel, &f.AllowedJobTypesJSON, &f.IngestionMode, &f.SyncMode, &f.Status, &f.SyncStatus,
		&f.ExternalRef, &f.KnowledgeScope, &notes, &f.ConnectorConfigJSON)
	if ownerTeam.Valid {
		u := uuid.UUID(ownerTeam.Bytes)
		f.OwnerTeamID = &u
	}
	if notes.Valid {
		s := notes.String
		f.Notes = &s
	}
	return err
}

func scanSourceFeedPublic(row pgx.Row, f *SourceFeed) error {
	var ownerTeam pgtype.UUID
	var notes pgtype.Text
	err := row.Scan(&f.ID, &f.ConnectorID, &f.DisplayName, &f.OwnerID, &ownerTeam, &f.DomainID,
		&f.SensitivityLevel, &f.AllowedJobTypesJSON, &f.IngestionMode, &f.SyncMode, &f.Status, &f.SyncStatus,
		&f.ExternalRef, &f.KnowledgeScope, &notes)
	if ownerTeam.Valid {
		u := uuid.UUID(ownerTeam.Bytes)
		f.OwnerTeamID = &u
	}
	if notes.Valid {
		s := notes.String
		f.Notes = &s
	}
	return err
}
