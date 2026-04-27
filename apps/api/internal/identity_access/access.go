package identity_access

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EvaluateInput struct {
	PrincipalID      uuid.UUID  `json:"principal_id"`
	Action           string     `json:"action"`
	ResourceType     string     `json:"resource_type"`
	ResourceID       *uuid.UUID `json:"resource_id,omitempty"`
	DomainID         *uuid.UUID `json:"domain_id,omitempty"`
	SensitivityLevel *int       `json:"sensitivity_level,omitempty"`
	// EntityType is set for entity resources to enforce access_policies.entity_type_scope when configured.
	EntityType *string `json:"entity_type,omitempty"`
	// ResourceOwnerID is optional context for future ABAC (owner-based rules); v1 evaluator does not branch on it.
	ResourceOwnerID *uuid.UUID `json:"resource_owner_id,omitempty"`
	// AccessPolicyID is optional when the target is explicitly bound to a policy row (future use).
	AccessPolicyID *uuid.UUID `json:"access_policy_id,omitempty"`
}

// AccessDecision is the outcome of the centralized permission pipeline.
// Trace is for internal logging only; do not return it verbatim to untrusted clients.
type AccessDecision struct {
	Allow            bool        `json:"allow"`
	ReasonCode       string      `json:"reason_code"`
	Reasons          []string    `json:"reasons"`
	Trace            []string    `json:"-"`
	MatchedPolicies  []uuid.UUID `json:"matched_policies,omitempty"`
	MatchedOverrides []uuid.UUID `json:"matched_overrides,omitempty"`
	SensitivityOK    bool        `json:"sensitivity_ok"`
	// EffectiveSensitivity is the resource sensitivity value that was evaluated (if any).
	EffectiveSensitivity *int       `json:"effective_sensitivity,omitempty"`
	ResolvedDomainID     *uuid.UUID `json:"resolved_domain_id,omitempty"`
	// SensitivityResult is a short audit-friendly summary of the sensitivity phase (empty if not reached).
	SensitivityResult string `json:"sensitivity_result,omitempty"`
	// MatchedRuleCode mirrors ReasonCode for admin/audit consumers that expect a "rule" field.
	MatchedRuleCode string `json:"matched_rule_code,omitempty"`
}

type AccessEvaluator struct {
	pool *pgxpool.Pool
}

func NewAccessEvaluator(pool *pgxpool.Pool) *AccessEvaluator {
	return &AccessEvaluator{pool: pool}
}

func levelAllowsAction(accessLevel, action string) bool {
	switch accessLevel {
	case "admin":
		return true
	case "write":
		switch action {
		case "view", "search", "view_raw", "edit", "create", "archive", "export",
			"run_job", "manage_jobs", "manage_sources", "manage_source_feed",
			"approve", "review", "publish":
			return true
		default:
			return false
		}
	case "read":
		switch action {
		case "view", "search", "view_raw":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// DomainIDsWithGrant returns domain IDs where the user has a non-expired domain_grants row.
func (e *AccessEvaluator) DomainIDsWithGrant(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if userID == uuid.Nil {
		return nil, nil
	}
	rows, err := e.pool.Query(ctx, `
		SELECT domain_id FROM domain_grants
		WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > now())`, userID)
	if err != nil {
		return nil, fmt.Errorf("access: domain grants: %w", err)
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var d uuid.UUID
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (e *AccessEvaluator) trace(dec *AccessDecision, msg string) {
	dec.Trace = append(dec.Trace, msg)
}

// Evaluate runs the permission resolution pipeline in a fixed order (documented in docs/permission-system.md):
//  1. Authenticated principal
//  2. Global deny rules
//  3. Policy overrides on the target (deny wins)
//  4. Entity ACL deny (and record allow for entity-type bypass)
//  5. Domain grant and domain scope
//  6. Entity-type policy (domain), with optional entity_acl allow bypass for view/search
//  7. Role action + domain access level for the requested action
//  8. Sensitivity cap vs resource sensitivity
//  9. Attach matched policies; allow
func (e *AccessEvaluator) Evaluate(ctx context.Context, in EvaluateInput) (*AccessDecision, error) {
	action := NormalizeAction(in.Action)
	dec := &AccessDecision{
		Allow:         false,
		ReasonCode:    ReasonDenyMissingPrincipal,
		Reasons:       []string{},
		Trace:         []string{},
		SensitivityOK: true,
	}
	if in.SensitivityLevel != nil {
		v := *in.SensitivityLevel
		dec.EffectiveSensitivity = &v
	}
	if in.DomainID != nil {
		d := *in.DomainID
		dec.ResolvedDomainID = &d
	}

	// 1. Principal
	if in.PrincipalID == uuid.Nil {
		e.trace(dec, "step:1_principal deny (nil)")
		dec.Reasons = append(dec.Reasons, "deny: missing principal")
		dec.MatchedRuleCode = dec.ReasonCode
		dec.SensitivityResult = "skipped"
		return dec, nil
	}
	e.trace(dec, "step:1_principal ok")

	// 2. Global deny
	blocked, reason, err := globalDenyForUser(ctx, e.pool, in.PrincipalID)
	if err != nil {
		return nil, fmt.Errorf("access: global deny: %w", err)
	}
	if blocked {
		dec.ReasonCode = ReasonDenyGlobalBlock
		dec.MatchedRuleCode = dec.ReasonCode
		dec.SensitivityResult = "skipped"
		dec.Reasons = append(dec.Reasons, "deny: global access block: "+reason)
		e.trace(dec, "step:2_global_deny")
		return dec, nil
	}
	e.trace(dec, "step:2_global_deny skip")

	// 3. Policy overrides
	ovrIDs, ovrTypes, err := loadOverrides(ctx, e.pool, in.ResourceType, in.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("access: overrides: %w", err)
	}
	for i, t := range ovrTypes {
		switch t {
		case "deny":
			dec.ReasonCode = ReasonDenyPolicyOverride
			dec.MatchedRuleCode = dec.ReasonCode
			dec.SensitivityResult = "skipped"
			dec.Reasons = append(dec.Reasons, "deny: policy override")
			dec.MatchedOverrides = append(dec.MatchedOverrides, ovrIDs[i])
			e.trace(dec, "step:3_policy_override deny")
			return dec, nil
		case "allow":
			dec.MatchedOverrides = append(dec.MatchedOverrides, ovrIDs[i])
		}
	}
	e.trace(dec, "step:3_policy_override ok")

	// 4. Entity ACL deny; detect allow row for type-policy bypass
	aclAllowBypass := false
	if in.ResourceType == "entity" && in.ResourceID != nil {
		denied, err := entityACLBlocksPrincipal(ctx, e.pool, *in.ResourceID, in.PrincipalID)
		if err != nil {
			return nil, fmt.Errorf("access: entity_acl: %w", err)
		}
		if denied {
			dec.ReasonCode = ReasonDenyEntityACL
			dec.MatchedRuleCode = dec.ReasonCode
			dec.SensitivityResult = "skipped"
			dec.Reasons = append(dec.Reasons, "deny: entity_acl")
			e.trace(dec, "step:4_entity_acl deny")
			return dec, nil
		}
		allowRow, err := entityACLAllowsPrincipal(ctx, e.pool, *in.ResourceID, in.PrincipalID)
		if err != nil {
			return nil, err
		}
		aclAllowBypass = allowRow && (action == "view" || action == "search")
		if allowRow {
			e.trace(dec, "step:4_entity_acl allow row present")
		} else {
			e.trace(dec, "step:4_entity_acl skip")
		}
	} else {
		e.trace(dec, "step:4_entity_acl n/a")
	}

	// 5. Domain scope
	if in.DomainID == nil {
		dec.ReasonCode = ReasonDenyDomainRequired
		dec.MatchedRuleCode = dec.ReasonCode
		dec.SensitivityResult = "skipped"
		dec.Reasons = append(dec.Reasons, "deny: domain scope required for evaluation")
		e.trace(dec, "step:5_domain missing domain_id")
		return dec, nil
	}

	var grantLevel string
	var cap int
	err = e.pool.QueryRow(ctx, `
		SELECT access_level, sensitivity_cap FROM domain_grants
		WHERE user_id = $1 AND domain_id = $2
		  AND (expires_at IS NULL OR expires_at > now())`,
		in.PrincipalID, *in.DomainID,
	).Scan(&grantLevel, &cap)
	if err != nil {
		dec.ReasonCode = ReasonDenyNoDomainGrant
		dec.MatchedRuleCode = dec.ReasonCode
		dec.SensitivityResult = "skipped"
		dec.Reasons = append(dec.Reasons, "deny: no domain grant")
		e.trace(dec, "step:5_domain no grant")
		return dec, nil
	}
	e.trace(dec, fmt.Sprintf("step:5_domain grant level=%s cap=%d", grantLevel, cap))

	// 6. Entity type (domain policy), bypass if entity ACL allow and view/search
	if in.ResourceType == "entity" && in.EntityType != nil && strings.TrimSpace(*in.EntityType) != "" {
		ok, err := entityTypeAllowedForDomain(ctx, e.pool, *in.DomainID, strings.TrimSpace(*in.EntityType))
		if err != nil {
			return nil, fmt.Errorf("access: entity_type_scope: %w", err)
		}
		if !ok && !(aclAllowBypass && (action == "view" || action == "search")) {
			dec.ReasonCode = ReasonDenyEntityType
			dec.MatchedRuleCode = dec.ReasonCode
			dec.SensitivityResult = "skipped"
			dec.Reasons = append(dec.Reasons, "deny: entity type not permitted by domain policy")
			e.trace(dec, "step:6_entity_type deny")
			return dec, nil
		}
		if !ok && aclAllowBypass {
			e.trace(dec, "step:6_entity_type bypass via entity_acl allow")
		} else {
			e.trace(dec, "step:6_entity_type ok")
		}
	} else {
		e.trace(dec, "step:6_entity_type n/a")
	}

	// 6b. Role-level entity type union (when assigned roles constrain types)
	if in.ResourceType == "entity" && in.EntityType != nil && strings.TrimSpace(*in.EntityType) != "" {
		ok, err := roleEntityTypeAllowedForAction(ctx, e.pool, in.PrincipalID, *in.DomainID, action, strings.TrimSpace(*in.EntityType))
		if err != nil {
			return nil, fmt.Errorf("access: role entity type scope: %w", err)
		}
		if !ok {
			dec.ReasonCode = ReasonDenyRoleEntityType
			dec.MatchedRuleCode = dec.ReasonCode
			dec.SensitivityResult = "skipped"
			dec.Reasons = append(dec.Reasons, "deny: entity type not permitted by role bindings")
			e.trace(dec, "step:6b_role_entity_type deny")
			return dec, nil
		}
		e.trace(dec, "step:6b_role_entity_type ok")
	} else {
		e.trace(dec, "step:6b_role_entity_type n/a")
	}

	// 7. Role + access level
	hasRoleAction, err := userHasRoleAction(ctx, e.pool, in.PrincipalID, *in.DomainID, action)
	if err != nil {
		return nil, fmt.Errorf("access: role action: %w", err)
	}
	if !hasRoleAction {
		dec.ReasonCode = ReasonDenyRoleAction
		dec.MatchedRuleCode = dec.ReasonCode
		dec.SensitivityResult = "skipped"
		dec.Reasons = append(dec.Reasons, "deny: role does not include action")
		e.trace(dec, "step:7_action role mismatch")
		return dec, nil
	}
	if !levelAllowsAction(grantLevel, action) {
		dec.ReasonCode = ReasonDenyAccessLevel
		dec.MatchedRuleCode = dec.ReasonCode
		dec.SensitivityResult = "skipped"
		dec.Reasons = append(dec.Reasons, "deny: domain access level insufficient for action")
		e.trace(dec, "step:7_action level mismatch")
		return dec, nil
	}
	e.trace(dec, "step:7_action ok")

	// 8. Sensitivity
	if in.SensitivityLevel != nil && *in.SensitivityLevel > cap {
		dec.SensitivityOK = false
		dec.ReasonCode = ReasonDenySensitivity
		dec.MatchedRuleCode = dec.ReasonCode
		dec.SensitivityResult = fmt.Sprintf("exceeds_cap(resource_level=%d,grant_cap=%d)", *in.SensitivityLevel, cap)
		dec.Reasons = append(dec.Reasons, "deny: sensitivity exceeds grant cap")
		e.trace(dec, "step:8_sensitivity deny")
		return dec, nil
	}
	e.trace(dec, "step:8_sensitivity ok")

	// 9. Policies + allow
	policyIDs, err := matchedPolicies(ctx, e.pool, *in.DomainID)
	if err != nil {
		return nil, fmt.Errorf("access: policies: %w", err)
	}
	dec.MatchedPolicies = policyIDs
	dec.Allow = true
	dec.ReasonCode = ReasonAllowOK
	dec.MatchedRuleCode = dec.ReasonCode
	if in.SensitivityLevel != nil {
		dec.SensitivityResult = fmt.Sprintf("within_cap(resource_level=%d,grant_cap=%d)", *in.SensitivityLevel, cap)
	} else {
		dec.SensitivityResult = "no_resource_level"
	}
	dec.Reasons = append(dec.Reasons, "allow: pipeline complete")
	e.trace(dec, "step:9_allow")
	return dec, nil
}

func globalDenyForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (blocked bool, reason string, err error) {
	var r string
	err = pool.QueryRow(ctx, `
		SELECT reason FROM global_access_denials WHERE user_id = $1 LIMIT 1`, userID).Scan(&r)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, r, nil
}

func loadOverrides(ctx context.Context, pool *pgxpool.Pool, resType string, resID *uuid.UUID) ([]uuid.UUID, []string, error) {
	if resID == nil {
		return nil, nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id, override_type FROM policy_overrides
		WHERE target_type = $1 AND target_id = $2
		  AND (expires_at IS NULL OR expires_at > now())
		ORDER BY created_at ASC`, resType, *resID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	var types []string
	for rows.Next() {
		var id uuid.UUID
		var t string
		if err := rows.Scan(&id, &t); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
		types = append(types, t)
	}
	return ids, types, rows.Err()
}

func userHasRoleAction(ctx context.Context, pool *pgxpool.Pool, userID, domainID uuid.UUID, action string) (bool, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM user_role_bindings urb
		JOIN roles r ON r.id = urb.role_id AND r.active = true
		JOIN role_action_permissions rap ON rap.role_id = urb.role_id
		JOIN action_permissions ap ON ap.id = rap.action_permission_id
		WHERE urb.user_id = $1
		  AND ap.code = $2
		  AND (urb.expires_at IS NULL OR urb.expires_at > now())
		  AND (
			urb.scope_type = 'global'
			OR (urb.scope_type = 'domain' AND urb.scope_id = $3)
		  )
		  AND (
			NOT EXISTS (SELECT 1 FROM role_domain_bindings rdb WHERE rdb.role_id = r.id)
			OR EXISTS (SELECT 1 FROM role_domain_bindings rdb WHERE rdb.role_id = r.id AND rdb.domain_id = $3)
		  )`, userID, action, domainID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// roleEntityTypeAllowedForAction returns true when no role constrains entity types for this action,
// or when entityType is in the union of types from applicable roles that declare bindings.
func roleEntityTypeAllowedForAction(ctx context.Context, pool *pgxpool.Pool, userID, domainID uuid.UUID, action, entityType string) (bool, error) {
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT ret.entity_type
		FROM user_role_bindings urb
		JOIN roles r ON r.id = urb.role_id AND r.active = true
		JOIN role_action_permissions rap ON rap.role_id = urb.role_id
		JOIN action_permissions ap ON ap.id = rap.action_permission_id
		JOIN role_entity_type_bindings ret ON ret.role_id = r.id
		WHERE urb.user_id = $1
		  AND ap.code = $2
		  AND (urb.expires_at IS NULL OR urb.expires_at > now())
		  AND (
			urb.scope_type = 'global'
			OR (urb.scope_type = 'domain' AND urb.scope_id = $3)
		  )
		  AND (
			NOT EXISTS (SELECT 1 FROM role_domain_bindings rdb WHERE rdb.role_id = r.id)
			OR EXISTS (SELECT 1 FROM role_domain_bindings rdb WHERE rdb.role_id = r.id AND rdb.domain_id = $3)
		  )`, userID, action, domainID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var allowed []string
	for rows.Next() {
		var et string
		if err := rows.Scan(&et); err != nil {
			return false, err
		}
		allowed = append(allowed, et)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	if len(allowed) == 0 {
		return true, nil
	}
	for _, et := range allowed {
		if et == entityType {
			return true, nil
		}
	}
	return false, nil
}

func matchedPolicies(ctx context.Context, pool *pgxpool.Pool, domainID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
		SELECT id FROM access_policies
		WHERE (domain_id IS NULL OR domain_id = $1) AND status = 'active'`, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
