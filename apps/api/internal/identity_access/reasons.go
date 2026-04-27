package identity_access

// ReasonCode is a stable machine identifier for access outcomes (audit, debugging, admin UI).
// HTTP handlers should not expose internal trace lines; use ReasonCode and high-level Reasons only where appropriate.
const (
	ReasonAllowOK = "ALLOW_OK"

	ReasonDenyMissingPrincipal = "DENY_MISSING_PRINCIPAL"
	ReasonDenyGlobalBlock      = "DENY_GLOBAL_BLOCK"
	ReasonDenyPolicyOverride   = "DENY_POLICY_OVERRIDE"
	ReasonDenyEntityACL        = "DENY_ENTITY_ACL"
	ReasonDenyDomainRequired   = "DENY_DOMAIN_REQUIRED"
	ReasonDenyNoDomainGrant    = "DENY_NO_DOMAIN_GRANT"
	ReasonDenyEntityType       = "DENY_ENTITY_TYPE_POLICY"
	ReasonDenyRoleAction       = "DENY_ROLE_ACTION"
	ReasonDenyRoleEntityType   = "DENY_ROLE_ENTITY_TYPE_SCOPE"
	ReasonDenyAccessLevel      = "DENY_ACCESS_LEVEL"
	ReasonDenySensitivity      = "DENY_SENSITIVITY_CAP"
)
