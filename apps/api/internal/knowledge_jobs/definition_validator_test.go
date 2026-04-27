package knowledge_jobs

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNormalizePublicationModeForStorage_legacyDraft(t *testing.T) {
	if got := NormalizePublicationModeForStorage("draft"); got != PublicationModeReviewedPublish {
		t.Fatalf("got %q", got)
	}
}

func TestValidateCreateJobInput_weeklyDigest_requiresSourceFeed(t *testing.T) {
	d := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	f := uuid.MustParse("21000000-0000-0000-0000-000000000001")
	scope, _ := json.Marshal(map[string]uuid.UUID{"domain_id": d})
	err := ValidateCreateJobInput(CreateJobInput{
		JobType:           "weekly_digest",
		SourceScopeJSON:   scope,
		PublicationMode:   PublicationModeReviewedPublish,
		OutputDomainID:    &d,
		ReviewRequired:    true,
		OutputSensitivity: 0,
	})
	if err == nil {
		t.Fatal("expected error")
	}

	scope2, _ := json.Marshal(map[string]uuid.UUID{"source_feed_id": f, "domain_id": d})
	if err := ValidateCreateJobInput(CreateJobInput{
		JobType:           "weekly_digest",
		SourceScopeJSON:   scope2,
		PublicationMode:   PublicationModeReviewedPublish,
		OutputDomainID:    &d,
		ReviewRequired:    true,
		OutputSensitivity: 0,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCreateJobInput_outputDomainWhenPublishing(t *testing.T) {
	d := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	f := uuid.MustParse("21000000-0000-0000-0000-000000000001")
	scope, _ := json.Marshal(map[string]uuid.UUID{"source_feed_id": f, "domain_id": d})
	err := ValidateCreateJobInput(CreateJobInput{
		JobType:           "weekly_digest",
		SourceScopeJSON:   scope,
		PublicationMode:   PublicationModeReviewedPublish,
		OutputDomainID:    nil,
		ReviewRequired:    true,
		OutputSensitivity: 0,
	})
	if !errors.Is(err, ErrOutputDomainRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateCreateJobInput_autoPublishWithReview(t *testing.T) {
	d := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	f := uuid.MustParse("21000000-0000-0000-0000-000000000001")
	scope, _ := json.Marshal(map[string]uuid.UUID{"source_feed_id": f, "domain_id": d})
	err := ValidateCreateJobInput(CreateJobInput{
		JobType:           "weekly_digest",
		SourceScopeJSON:   scope,
		PublicationMode:   PublicationModeAutoPublish,
		OutputDomainID:    &d,
		ReviewRequired:    true,
		OutputSensitivity: 0,
	})
	if !errors.Is(err, ErrPublicationModeReviewConflict) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateTriggerInput_scheduled(t *testing.T) {
	s := "0 9 * * *"
	if err := ValidateTriggerInput(CreateTriggerInput{TriggerType: "scheduled", ScheduleExpr: &s}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTriggerInput(CreateTriggerInput{TriggerType: "scheduled"}); !errors.Is(err, ErrTriggerScheduledMissingSchedule) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateTriggerInput_eventDriven(t *testing.T) {
	err := ValidateTriggerInput(CreateTriggerInput{
		TriggerType:     "event_driven",
		EventFilterJSON: json.RawMessage(`{}`),
	})
	if !errors.Is(err, ErrTriggerEventFilterRequired) {
		t.Fatalf("got %v", err)
	}
	if err := ValidateTriggerInput(CreateTriggerInput{
		TriggerType:     "event_driven",
		EventFilterJSON: json.RawMessage(`{"connector":"x"}`),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestIsClientJobDefinitionError(t *testing.T) {
	if !IsClientJobDefinitionError(ErrSourceScopeRequired) {
		t.Fatal("expected true")
	}
	if !IsClientJobDefinitionError(ErrUnimplementedJobType) {
		t.Fatal("expected true for ErrUnimplementedJobType")
	}
	if IsClientJobDefinitionError(errors.New("other")) {
		t.Fatal("expected false")
	}
}

func TestValidateCreateJobInput_unrestrictedScopeRejected(t *testing.T) {
	d := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	f := uuid.MustParse("21000000-0000-0000-0000-000000000001")
	scope, _ := json.Marshal(map[string]any{"unrestricted": true, "domain_id": d.String(), "source_feed_id": f.String()})
	err := ValidateCreateJobInput(CreateJobInput{
		JobType:           "weekly_digest",
		SourceScopeJSON:   scope,
		PublicationMode:   PublicationModeDraftOnly,
		ReviewRequired:    false,
		OutputSensitivity: 0,
	})
	if !errors.Is(err, ErrSourceScopeUnrestricted) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateCreateJobInput_unimplementedJobTypeRejected(t *testing.T) {
	d := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	scope, _ := json.Marshal(map[string]uuid.UUID{"domain_id": d})
	err := ValidateCreateJobInput(CreateJobInput{
		JobType:           "incident_summary",
		SourceScopeJSON:   scope,
		PublicationMode:   PublicationModeDraftOnly,
		ReviewRequired:    false,
		OutputSensitivity: 0,
	})
	if !errors.Is(err, ErrUnimplementedJobType) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateCreateJobInput_planningSummaryRequiresSourceFeed(t *testing.T) {
	d := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	f := uuid.MustParse("21000000-0000-0000-0000-000000000001")
	scopeMissingFeed, _ := json.Marshal(map[string]uuid.UUID{"domain_id": d})
	err := ValidateCreateJobInput(CreateJobInput{
		JobType:           "planning_summary",
		SourceScopeJSON:   scopeMissingFeed,
		PublicationMode:   PublicationModeDraftOnly,
		ReviewRequired:    false,
		OutputSensitivity: 0,
	})
	if err == nil {
		t.Fatal("expected source_feed_id requirement error")
	}

	scopeOK, _ := json.Marshal(map[string]uuid.UUID{"domain_id": d, "source_feed_id": f})
	if err := ValidateCreateJobInput(CreateJobInput{
		JobType:           "planning_summary",
		SourceScopeJSON:   scopeOK,
		PublicationMode:   PublicationModeDraftOnly,
		ReviewRequired:    false,
		OutputSensitivity: 0,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCreateJobInput_supportTrendsRequiresSourceFeed(t *testing.T) {
	d := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	f := uuid.MustParse("21000000-0000-0000-0000-000000000001")
	scopeOK, _ := json.Marshal(map[string]uuid.UUID{"domain_id": d, "source_feed_id": f})
	if err := ValidateCreateJobInput(CreateJobInput{
		JobType:           "support_trends_extraction",
		SourceScopeJSON:   scopeOK,
		PublicationMode:   PublicationModeDraftOnly,
		ReviewRequired:    false,
		OutputSensitivity: 0,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateTriggerRowsForPrimaryType_mismatch(t *testing.T) {
	s := "0 9 * * *"
	err := ValidateTriggerRowsForPrimaryType("scheduled", []JobTriggerRow{
		{TriggerType: "event_driven", Status: "active"},
	})
	if !errors.Is(err, ErrTriggerRowsMismatch) {
		t.Fatalf("got %v", err)
	}
	err = ValidateTriggerRowsForPrimaryType("scheduled", []JobTriggerRow{
		{TriggerType: "scheduled", Status: "active", ScheduleExpr: &s},
	})
	if err != nil {
		t.Fatal(err)
	}
}
