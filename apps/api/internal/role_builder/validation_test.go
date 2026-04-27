package role_builder

import "testing"

func TestValidateWriteInput_CreateRequiresActions(t *testing.T) {
	err := ValidateWriteInput(RoleWriteInput{Code: "x", Name: "X", ActionCodes: []string{}}, true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateWriteInput_OverrideRequiresDomainBindings(t *testing.T) {
	err := ValidateWriteInput(RoleWriteInput{
		Code: "x", Name: "X", ScopeModel: "domain",
		ActionCodes: []string{"view"},
		Governance:  &GovernanceRow{CanOverridePolicies: true},
	}, true)
	if err == nil {
		t.Fatal("expected error for override without domain bindings")
	}
}

func TestValidateReplaceBindings(t *testing.T) {
	err := ValidateReplaceBindings(RoleWriteInput{Code: "x", Name: "X", ActionCodes: []string{}})
	if err == nil {
		t.Fatal("expected error")
	}
}
