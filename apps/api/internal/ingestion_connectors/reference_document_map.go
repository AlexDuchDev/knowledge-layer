package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/knowledge_core"
)

// MapNormalizedRecordToReferenceDocument creates a ReferenceDocument entity from a google_drive_document
// normalized record, attaches provenance to raw + normalized sources, and does not publish or canonicalize.
func MapNormalizedRecordToReferenceDocument(ctx context.Context, ing *Service, entities *knowledge_core.EntityRepo, normID, actor uuid.UUID, truthMode string) (*knowledge_core.Entity, error) {
	nrec, err := ing.GetNormalizedRecord(ctx, normID)
	if err != nil {
		return nil, err
	}
	if nrec.RecordType != RecordTypeGoogleDriveDocument {
		return nil, fmt.Errorf("record type %q cannot map to ReferenceDocument (expected %s)", nrec.RecordType, RecordTypeGoogleDriveDocument)
	}

	var payload struct {
		Title       string `json:"title"`
		BodyText    string `json:"body_text"`
		ExternalRef string `json:"external_ref"`
		WebViewLink string `json:"web_view_link,omitempty"`
	}
	if err := json.Unmarshal(nrec.StructuredPayloadJSON, &payload); err != nil {
		return nil, fmt.Errorf("payload: %w", err)
	}
	if payload.ExternalRef == "" {
		return nil, errors.New("normalized record missing external_ref")
	}
	if payload.Title == "" {
		return nil, errors.New("normalized record missing title")
	}

	feed, err := ing.GetSourceFeed(ctx, nrec.SourceFeedID)
	if err != nil {
		return nil, err
	}
	ownerID := feed.OwnerID
	if ownerID == uuid.Nil {
		return nil, errors.New("source feed has no owner")
	}

	if truthMode == "" {
		truthMode = "mirrored_authority"
	}

	payloadJSON, _ := json.Marshal(map[string]any{
		"source":               "google_drive_ingestion",
		"normalized_record_id": normID.String(),
		"web_view_link":        payload.WebViewLink,
	})

	body := payload.BodyText
	ent, err := entities.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "ReferenceDocument",
		Title:            payload.Title,
		Body:             &body,
		OwnerID:          &ownerID,
		DomainID:         feed.DomainID,
		SensitivityLevel: feed.SensitivityLevel,
		TruthMode:        truthMode,
		LifecycleState:   "draft",
		PayloadJSON:      payloadJSON,
		ExternalRef:      &payload.ExternalRef,
	})
	if err != nil {
		return nil, err
	}

	fid := feed.ID
	extRef := payload.ExternalRef
	_, err = entities.AttachProvenance(ctx, knowledge_core.ProvenanceRecord{
		TargetType:   "entity",
		TargetID:     ent.ID,
		OriginType:   "google_drive_file",
		OriginRef:    &extRef,
		SourceFeedID: &fid,
		CreatedByID:  &actor,
	}, []uuid.UUID{nrec.RawArtifactID}, []uuid.UUID{normID})
	if err != nil {
		return nil, fmt.Errorf("provenance: %w", err)
	}

	return ent, nil
}
