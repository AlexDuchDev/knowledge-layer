package knowledge_jobs

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestApplyJobTemplate_unknown(t *testing.T) {
	owner := uuid.New()
	in := CreateJobInput{TemplateID: "nope", OwnerID: owner}
	_, err := ApplyJobTemplate(in)
	if !errors.Is(err, ErrUnknownJobTemplate) {
		t.Fatalf("expected ErrUnknownJobTemplate, got %v", err)
	}
}

func TestApplyJobTemplate_weeklyDigest_defaultsAndMerge(t *testing.T) {
	owner := uuid.New()
	in := CreateJobInput{
		TemplateID:      "weekly_digest",
		OwnerID:         owner,
		SourceScopeJSON: json.RawMessage(`{"domain_id":"32000000-0000-0000-0000-000000000001","source_feed_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`),
		ConfigJSON:      json.RawMessage(`{"window_days":14}`),
	}
	out, err := ApplyJobTemplate(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "Weekly digest" {
		t.Fatalf("name: %q", out.Name)
	}
	if out.JobType != "weekly_digest" {
		t.Fatalf("job type: %q", out.JobType)
	}
	if !out.ReviewRequired || out.OutputSensitivity != 0 {
		t.Fatalf("review/sens: %v %d", out.ReviewRequired, out.OutputSensitivity)
	}
	if out.Purpose == nil || *out.Purpose == "" {
		t.Fatal("expected purpose from template")
	}
	var cfg map[string]any
	if err := json.Unmarshal(out.ConfigJSON, &cfg); err != nil {
		t.Fatal(err)
	}
	// overlay wins for shared keys
	if int(cfg["window_days"].(float64)) != 14 {
		t.Fatalf("window_days merge: %#v", cfg["window_days"])
	}
}

func TestApplyJobTemplate_configMerge_preservesBaseKeys(t *testing.T) {
	in := CreateJobInput{
		TemplateID: "leadership_brief",
		OwnerID:    uuid.New(),
		ConfigJSON: json.RawMessage(`{"brief_owner":"exec"}`),
	}
	out, err := ApplyJobTemplate(in)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(out.ConfigJSON, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["format"] != "brief" {
		t.Fatalf("expected format from template: %#v", cfg["format"])
	}
	if cfg["brief_owner"] != "exec" {
		t.Fatalf("expected overlay key: %#v", cfg["brief_owner"])
	}
}

func TestApplyJobTemplate_nameOverride(t *testing.T) {
	owner := uuid.New()
	in := CreateJobInput{
		TemplateID: "planning_summary",
		Name:       "Q1 horizon",
		OwnerID:    owner,
	}
	out, err := ApplyJobTemplate(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "Q1 horizon" {
		t.Fatalf("name: %q", out.Name)
	}
	if out.JobType != "planning_summary" {
		t.Fatalf("job type: %q", out.JobType)
	}
}
