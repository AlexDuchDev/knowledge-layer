package ingestion_connectors

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestValidateCreateSourceFeedInput_RequiresGovernanceFields(t *testing.T) {
	base := CreateSourceFeedInput{
		ConnectorID:         uuid.New(),
		DisplayName:         "x",
		OwnerID:             uuid.New(),
		DomainID:            uuid.New(),
		SensitivityLevel:    0,
		AllowedJobTypesJSON: json.RawMessage(`["weekly_digest"]`),
		IngestionMode:       "ingestion_only",
		SyncMode:            SyncModeManual,
		KnowledgeScope:      "domain_linked",
		ConnectorConfigJSON: json.RawMessage(`{}`),
	}
	if err := ValidateCreateSourceFeedInput(base); err != nil {
		t.Fatal(err)
	}

	noDomain := base
	noDomain.DomainID = uuid.Nil
	if err := ValidateCreateSourceFeedInput(noDomain); err == nil {
		t.Fatal("expected error without domain_id")
	}

	noJobs := base
	noJobs.AllowedJobTypesJSON = json.RawMessage(`[]`)
	if err := ValidateCreateSourceFeedInput(noJobs); err == nil {
		t.Fatal("expected error for empty allowed jobs")
	}

	badSync := base
	badSync.SyncMode = "nope"
	if err := ValidateCreateSourceFeedInput(badSync); err == nil {
		t.Fatal("expected error for invalid sync_mode")
	}
}
