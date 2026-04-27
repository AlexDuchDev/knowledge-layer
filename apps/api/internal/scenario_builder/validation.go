package scenario_builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrMissingName         = errors.New("scenario name required")
	ErrMissingCode         = errors.New("scenario code required")
	ErrMissingType         = errors.New("scenario_type required")
	ErrMissingOutputMode   = errors.New("output_mode required")
	ErrMissingInputScope   = errors.New("input_scope required for this scenario type")
	ErrMissingOutputPolicy = errors.New("output policy required for this scenario type")
	ErrUnrestrictedInput   = errors.New("unrestricted input_scope not allowed without config allow_unrestricted_input_scope")
	ErrInvalidTrigger      = errors.New("trigger_config invalid for trigger_type")
)

// ValidateWriteInput validates create / full replace.
func ValidateWriteInput(in ScenarioWriteInput, isCreate bool) error {
	if strings.TrimSpace(in.Name) == "" {
		return ErrMissingName
	}
	if strings.TrimSpace(in.Code) == "" {
		return ErrMissingCode
	}
	if strings.TrimSpace(in.ScenarioType) == "" {
		return ErrMissingType
	}
	if strings.TrimSpace(in.OutputMode) == "" {
		return ErrMissingOutputMode
	}
	st := strings.TrimSpace(in.ScenarioType)
	if !validScenarioType(st) {
		return fmt.Errorf("invalid scenario_type: %s", st)
	}
	if err := validateInputScope(st, in.InputScopeJSON, in.ConfigJSON); err != nil {
		return err
	}
	if err := validateOutputPolicyRequired(st, in.OutputPolicy); err != nil {
		return err
	}
	if err := validateTrigger(in.TriggerType, in.TriggerConfigJSON); err != nil {
		return err
	}
	return nil
}

func validScenarioType(s string) bool {
	switch s {
	case "ask", "digest", "process", "explorer", "governance":
		return true
	default:
		return false
	}
}

func requiresOutputPolicy(scenarioType string) bool {
	switch scenarioType {
	case "digest", "process", "governance", "ask":
		return true
	default:
		return false
	}
}

func validateOutputPolicyRequired(scenarioType string, pol *OutputPolicyWrite) error {
	if !requiresOutputPolicy(scenarioType) {
		return nil
	}
	if pol == nil {
		return ErrMissingOutputPolicy
	}
	return nil
}

func validateInputScope(scenarioType string, inputScope, configJSON json.RawMessage) error {
	var scope map[string]json.RawMessage
	if len(inputScope) > 0 && string(inputScope) != "null" {
		if err := json.Unmarshal(inputScope, &scope); err != nil {
			return fmt.Errorf("input_scope_json: %w", err)
		}
	} else {
		scope = map[string]json.RawMessage{}
	}

	var cfg map[string]json.RawMessage
	if len(configJSON) > 0 && string(configJSON) != "null" {
		_ = json.Unmarshal(configJSON, &cfg)
	}

	// Unrestricted corpus guard
	if u, ok := scope["unrestricted"]; ok {
		var b bool
		if json.Unmarshal(u, &b) == nil && b {
			allow := false
			if cfg != nil {
				if a, ok := cfg["allow_unrestricted_input_scope"]; ok {
					_ = json.Unmarshal(a, &allow)
				}
			}
			if !allow {
				return ErrUnrestrictedInput
			}
		}
	}

	switch scenarioType {
	case "digest", "process", "governance":
		if len(scope) == 0 {
			return ErrMissingInputScope
		}
		if !inputScopeExplicit(scope, scenarioType) {
			return ErrMissingInputScope
		}
	case "ask":
		if len(scope) == 0 {
			return ErrMissingInputScope
		}
		if !scopeHasInheritRetrieval(scope) && !inputScopeExplicit(scope, scenarioType) {
			return ErrMissingInputScope
		}
	case "explorer":
		if len(scope) == 0 {
			return ErrMissingInputScope
		}
		if !inputScopeExplicit(scope, scenarioType) {
			return ErrMissingInputScope
		}
	}
	return nil
}

func scopeHasInheritRetrieval(scope map[string]json.RawMessage) bool {
	v, ok := scope["inherit_user_retrieval_scope"]
	if !ok {
		return false
	}
	var b bool
	return json.Unmarshal(v, &b) == nil && b
}

func inputScopeExplicit(scope map[string]json.RawMessage, scenarioType string) bool {
	if scopeHasInheritRetrieval(scope) {
		return true
	}
	if _, ok := scope["requires_explicit_scope"]; ok {
		return true
	}
	if _, ok := scope["domain_ids"]; ok {
		return true
	}
	if _, ok := scope["source_feed_ids"]; ok {
		return true
	}
	if _, ok := scope["entity_types"]; ok {
		return true
	}
	if _, ok := scope["projects"]; ok {
		return true
	}
	if _, ok := scope["time_window"]; ok {
		return true
	}
	if _, ok := scope["source_categories"]; ok {
		return true
	}
	if _, ok := scope["governance_views"]; ok {
		return scenarioType == "governance"
	}
	return false
}

func validateTrigger(triggerType string, triggerConfig json.RawMessage) error {
	tt := strings.TrimSpace(triggerType)
	var cfg map[string]json.RawMessage
	if len(triggerConfig) > 0 && string(triggerConfig) != "null" {
		if err := json.Unmarshal(triggerConfig, &cfg); err != nil {
			return fmt.Errorf("trigger_config_json: %w", err)
		}
	} else {
		cfg = map[string]json.RawMessage{}
	}
	switch tt {
	case "interactive", "manual":
		return nil
	case "scheduled":
		if _, ok := cfg["schedule_expr"]; !ok {
			return fmt.Errorf("%w: scheduled requires schedule_expr", ErrInvalidTrigger)
		}
		return nil
	case "event_driven":
		if et, ok := cfg["event_types"]; ok {
			var arr []string
			if json.Unmarshal(et, &arr) == nil && len(arr) > 0 {
				return nil
			}
		}
		if _, ok := cfg["event_filter"]; ok {
			return nil
		}
		return fmt.Errorf("%w: event_driven requires event_types or event_filter", ErrInvalidTrigger)
	case "conditional":
		if _, ok := cfg["condition"]; !ok {
			if _, ok2 := cfg["condition_expr"]; !ok2 {
				return fmt.Errorf("%w: conditional requires condition or condition_expr", ErrInvalidTrigger)
			}
		}
		return nil
	default:
		return fmt.Errorf("invalid trigger_type: %s", tt)
	}
}

// ValidatePatchBindings checks replace binding payloads reference real rows (caller passes existence flags).
func ValidateRoleBindingWrites(rows []RoleBindingWrite) error {
	for _, r := range rows {
		if r.RoleID == uuid.Nil {
			return errors.New("role_id required in role binding")
		}
	}
	return nil
}
