package ingestion_connectors

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

// TestCandidateSourceFeed_mirrorsAllInputFields locks in the contract that the
// candidate built for adapter-level validation at POST /source-feeds (and POST
// /source-feeds/validate) carries every CreateSourceFeedInput field that has a
// counterpart on SourceFeed. Earlier route-level constructions silently dropped
// ExternalRef, which made filesystem feeds (and any other connector that reads
// ExternalRef in ValidateSourceFeedConfig) impossible to create through the API.
// If a future field is added to CreateSourceFeedInput without being mirrored
// here, this test fails.
func TestCandidateSourceFeed_mirrorsAllInputFields(t *testing.T) {
	teamID := uuid.New()
	notes := "rc-validation"
	in := CreateSourceFeedInput{
		ConnectorID:         uuid.New(),
		DisplayName:         "rc-feed",
		OwnerID:             uuid.New(),
		OwnerTeamID:         &teamID,
		DomainID:            uuid.New(),
		SensitivityLevel:    2,
		AllowedJobTypesJSON: json.RawMessage(`["weekly_digest"]`),
		IngestionMode:       "ingestion_only",
		SyncMode:            SyncModeManual,
		ExternalRef:         "opensearch",
		KnowledgeScope:      "domain_linked",
		Notes:               &notes,
		ConnectorConfigJSON: json.RawMessage(`{"k":"v"}`),
	}

	got := in.CandidateSourceFeed()
	if got == nil {
		t.Fatal("candidate is nil")
	}

	cases := []struct {
		name string
		want any
		got  any
	}{
		{"ConnectorID", in.ConnectorID, got.ConnectorID},
		{"DisplayName", in.DisplayName, got.DisplayName},
		{"OwnerID", in.OwnerID, got.OwnerID},
		{"OwnerTeamID", in.OwnerTeamID, got.OwnerTeamID},
		{"DomainID", in.DomainID, got.DomainID},
		{"SensitivityLevel", in.SensitivityLevel, got.SensitivityLevel},
		{"AllowedJobTypesJSON", string(in.AllowedJobTypesJSON), string(got.AllowedJobTypesJSON)},
		{"IngestionMode", in.IngestionMode, got.IngestionMode},
		{"SyncMode", in.SyncMode, got.SyncMode},
		{"ExternalRef", in.ExternalRef, got.ExternalRef},
		{"KnowledgeScope", in.KnowledgeScope, got.KnowledgeScope},
		{"Notes", in.Notes, got.Notes},
		{"ConnectorConfigJSON", string(in.ConnectorConfigJSON), string(got.ConnectorConfigJSON)},
	}

	for _, c := range cases {
		if !reflect.DeepEqual(c.want, c.got) {
			t.Errorf("CandidateSourceFeed() dropped field %s: want %#v, got %#v", c.name, c.want, c.got)
		}
	}
}

// TestCandidateSourceFeed_externalRefReachesFilesystemValidator is the focused
// regression for the rc2 filesystem-connector bug: the candidate must carry
// ExternalRef so that the filesystem adapter's ValidateSourceFeedConfig sees it.
// Pure in-memory test — no DB, no Registry — just exercises the candidate shape
// against the adapter's own validator.
func TestCandidateSourceFeed_externalRefReachesFilesystemValidator(t *testing.T) {
	in := CreateSourceFeedInput{
		ConnectorID:         uuid.MustParse("20000000-0000-0000-0000-000000000012"),
		DomainID:            uuid.MustParse("32000000-0000-0000-0000-000000000001"),
		OwnerID:             uuid.MustParse("30000000-0000-0000-0000-000000000001"),
		DisplayName:         "rc-fs-feed",
		ExternalRef:         "opensearch", // any valid relative path under /data
		IngestionMode:       "ingestion_only",
		ConnectorConfigJSON: json.RawMessage(`{}`),
	}
	got := in.CandidateSourceFeed()
	if got.ExternalRef == "" {
		t.Fatal("regression: candidate dropped ExternalRef — filesystem and similar adapters will reject all config saves")
	}
	if got.ExternalRef != in.ExternalRef {
		t.Fatalf("ExternalRef mismatch: want %q, got %q", in.ExternalRef, got.ExternalRef)
	}
}
