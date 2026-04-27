package scenario_builder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/db"
)

func TestDefinitionService_CreateAndPresetClone(t *testing.T) {
	if os.Getenv("E2E_DB") == "" {
		t.Skip("set E2E_DB=1 and DATABASE_URL")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL required")
	}
	ctx := context.Background()
	if err := db.MigrateUp(dsn); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	svc := NewServices(pool)
	code := fmt.Sprintf("sc_test_%s", uuid.NewString()[:8])
	id, err := svc.Definitions.Create(ctx, ScenarioWriteInput{
		Code:              code,
		Name:              "Test scenario",
		ScenarioType:      "explorer",
		InputScopeJSON:    json.RawMessage(`{"entity_types":["decision"],"requires_explicit_scope":true}`),
		TriggerType:       "interactive",
		TriggerConfigJSON: json.RawMessage(`{}`),
		ProcessingMode:    "explore",
		OutputMode:        "explorer_view",
		UISurface:         "knowledge",
		OutputPolicy: &OutputPolicyWrite{
			PublicationMode:    "draft",
			ProvenanceRequired: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == uuid.Nil {
		t.Fatal("expected id")
	}

	cloneCode := fmt.Sprintf("sc_clone_%s", uuid.NewString()[:8])
	cloneID, err := svc.Presets.CreateFromPreset(ctx, FromPresetInput{
		PresetKey: "weekly_team_digest",
		Code:      cloneCode,
		Name:      "Cloned weekly digest",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cloneID == uuid.Nil {
		t.Fatal("expected clone id")
	}
	full, err := svc.Definitions.Get(ctx, cloneID)
	if err != nil {
		t.Fatal(err)
	}
	if full.ScenarioType != "digest" {
		t.Fatalf("type %s", full.ScenarioType)
	}
}
