package role_builder

import (
	"fmt"
	"strings"
)

var allowedScopeModels = map[string]struct{}{
	"global": {}, "domain": {}, "team": {}, "project": {}, "source": {},
}

// ValidateWriteInput checks role builder invariants before persistence.
func ValidateWriteInput(in RoleWriteInput, isCreate bool) error {
	if strings.TrimSpace(in.Code) == "" {
		return fmt.Errorf("code is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	sm := in.ScopeModel
	if sm == "" {
		sm = "global"
	}
	if _, ok := allowedScopeModels[sm]; !ok {
		return fmt.Errorf("invalid scope_model %q", sm)
	}

	gov := GovernanceRow{}
	if in.Governance != nil {
		gov = *in.Governance
	}
	if gov.CanOverridePolicies {
		if sm != "global" && sm != "domain" {
			return fmt.Errorf("can_override_policies requires scope_model global or domain")
		}
		if sm == "domain" && len(in.DomainIDs) == 0 {
			return fmt.Errorf("can_override_policies with domain scope_model requires at least one allowed domain binding")
		}
	}

	if isCreate && len(in.ActionCodes) == 0 {
		return fmt.Errorf("at least one action permission is required")
	}

	return nil
}

// ValidateReplaceBindings runs validation for full binding replacement on PATCH.
func ValidateReplaceBindings(in RoleWriteInput) error {
	if len(in.ActionCodes) == 0 {
		return fmt.Errorf("at least one action permission is required when replacing bindings")
	}
	return ValidateWriteInput(in, false)
}
