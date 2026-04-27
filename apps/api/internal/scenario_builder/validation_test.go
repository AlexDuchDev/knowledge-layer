package scenario_builder

import (
	"encoding/json"
	"testing"
)

func TestValidateWriteInput_MissingFields(t *testing.T) {
	err := ValidateWriteInput(ScenarioWriteInput{}, true)
	if err == nil {
		t.Fatal("expected error")
	}
	err = ValidateWriteInput(ScenarioWriteInput{Name: "N", Code: "c", ScenarioType: "ask"}, true)
	if err == nil {
		t.Fatal("expected error for missing output_mode")
	}
}

func TestValidateWriteInput_GovernanceWithoutPolicy(t *testing.T) {
	in := ScenarioWriteInput{
		Name: "x", Code: "x", ScenarioType: "governance",
		OutputMode:     "review_task",
		InputScopeJSON: json.RawMessage(`{"governance_views":["stale_content"],"requires_explicit_scope":true}`),
		TriggerType:    "interactive", TriggerConfigJSON: json.RawMessage(`{}`),
		ProcessingMode: "explore",
	}
	err := ValidateWriteInput(in, true)
	if err != ErrMissingOutputPolicy {
		t.Fatalf("got %v want ErrMissingOutputPolicy", err)
	}
}

func TestValidateWriteInput_UnrestrictedInput(t *testing.T) {
	in := ScenarioWriteInput{
		Name: "x", Code: "x", ScenarioType: "ask",
		OutputMode: "ui_response", OutputPolicy: &OutputPolicyWrite{PublicationMode: "draft"},
		InputScopeJSON: json.RawMessage(`{"unrestricted":true}`),
		TriggerType:    "interactive", TriggerConfigJSON: json.RawMessage(`{}`),
		ProcessingMode: "ask",
	}
	err := ValidateWriteInput(in, true)
	if err != ErrUnrestrictedInput {
		t.Fatalf("got %v", err)
	}
	in.ConfigJSON = json.RawMessage(`{"allow_unrestricted_input_scope":true}`)
	in.InputScopeJSON = json.RawMessage(`{"unrestricted":true,"inherit_user_retrieval_scope":true}`)
	err = ValidateWriteInput(in, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateWriteInput_TriggerScheduled(t *testing.T) {
	in := ScenarioWriteInput{
		Name: "x", Code: "x", ScenarioType: "digest",
		OutputMode: "digest_entity", OutputPolicy: &OutputPolicyWrite{PublicationMode: "draft"},
		InputScopeJSON: json.RawMessage(`{"time_window":"last_7d","requires_explicit_scope":true}`),
		TriggerType:    "scheduled", TriggerConfigJSON: json.RawMessage(`{}`),
		ProcessingMode: "summarize",
	}
	err := ValidateWriteInput(in, true)
	if err == nil {
		t.Fatal("expected invalid trigger")
	}
	in.TriggerConfigJSON = json.RawMessage(`{"schedule_expr":"0 9 * * *"}`)
	if err := ValidateWriteInput(in, true); err != nil {
		t.Fatal(err)
	}
}

func TestValidateWriteInput_DigestEmptyScope(t *testing.T) {
	in := ScenarioWriteInput{
		Name: "x", Code: "x", ScenarioType: "digest",
		OutputMode: "x", OutputPolicy: &OutputPolicyWrite{},
		InputScopeJSON: json.RawMessage(`{}`),
		TriggerType:    "manual", TriggerConfigJSON: json.RawMessage(`{}`),
	}
	err := ValidateWriteInput(in, true)
	if err != ErrMissingInputScope {
		t.Fatalf("got %v", err)
	}
}
