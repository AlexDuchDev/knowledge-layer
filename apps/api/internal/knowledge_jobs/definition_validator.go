package knowledge_jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Publication modes (canonical). Legacy DB value "draft" on publication_mode meant review pipeline — migrate to reviewed_publish.
const (
	PublicationModeDraftOnly         = "draft_only"
	PublicationModeReviewedPublish   = "reviewed_publish"
	PublicationModeAutoPublish       = "auto_publish"
	legacyPublicationModeColumnDraft = "draft"
)

var (
	// ErrSourceScopeUnrestricted is returned when unrestricted broad scope is not allowed.
	ErrSourceScopeUnrestricted = errors.New("unrestricted source scope is not permitted for this job type")
	// ErrTriggerRowsMismatch is returned when job trigger_type requires a matching active job_triggers row.
	ErrTriggerRowsMismatch = errors.New("job trigger_type requires at least one matching active job_triggers row")
	// ErrProcessingModeInvalid is returned when processing_mode is not in the allowed set.
	ErrProcessingModeInvalid = errors.New("invalid processing_mode")
	// ErrInvalidPublicationMode is returned when publication_mode is not allowed.
	ErrInvalidPublicationMode = errors.New("invalid publication_mode")
	// ErrSourceScopeRequired is returned when source_scope_json is empty or not an object.
	ErrSourceScopeRequired = errors.New("source_scope_json must be a non-empty object with governed scope")
	// ErrSourceScopeInvalid is returned when source_scope_json is not valid JSON.
	ErrSourceScopeInvalid = errors.New("source_scope_json must be valid JSON")
	// ErrOutputDomainRequired is returned when output_domain_id is required by publication policy.
	ErrOutputDomainRequired = errors.New("output_domain_id is required for this publication_mode")
	// ErrPublicationModeReviewConflict is returned when auto_publish conflicts with review_required.
	ErrPublicationModeReviewConflict = errors.New("publication_mode auto_publish is not allowed when review_required is true")
)

// NormalizePublicationMode lowercases known modes; unknown strings are returned trimmed for validation errors.
func NormalizePublicationMode(s string) string {
	t := strings.TrimSpace(strings.ToLower(s))
	switch t {
	case PublicationModeDraftOnly, PublicationModeReviewedPublish, PublicationModeAutoPublish:
		return t
	default:
		return strings.TrimSpace(s)
	}
}

// NormalizePublicationModeForStorage maps API/legacy aliases to a canonical value persisted in DB.
func NormalizePublicationModeForStorage(s string) string {
	t := strings.TrimSpace(strings.ToLower(s))
	if t == legacyPublicationModeColumnDraft {
		return PublicationModeReviewedPublish
	}
	return NormalizePublicationMode(s)
}

// ValidatePublicationMode returns an error if mode is not one of the canonical values (after normalization check).
func ValidatePublicationMode(mode string) error {
	n := NormalizePublicationModeForStorage(mode)
	switch NormalizePublicationMode(n) {
	case PublicationModeDraftOnly, PublicationModeReviewedPublish, PublicationModeAutoPublish:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidPublicationMode, mode)
	}
}

// AllowedProcessingModes lists canonical processing_mode values (matches DB check).
var AllowedProcessingModes = map[string]struct{}{
	"summarize": {}, "extract": {}, "consolidate": {}, "detect": {}, "transform": {}, "publish": {},
}

// ValidateProcessingMode returns an error if mode is not allowed.
func ValidateProcessingMode(mode string) error {
	m := strings.TrimSpace(strings.ToLower(mode))
	if m == "" {
		return fmt.Errorf("%w: empty", ErrProcessingModeInvalid)
	}
	if _, ok := AllowedProcessingModes[m]; !ok {
		return fmt.Errorf("%w: %q", ErrProcessingModeInvalid, mode)
	}
	return nil
}

// ValidateCreateJobInput checks definition invariants after template merge and JSON defaults.
// New jobs must use a job_type with a runtime processor (fail-closed).
func ValidateCreateJobInput(in CreateJobInput) error {
	return validateJobDefinitionInput(in, true)
}

// ValidateUpdateJobInput validates a merged job definition (after Patch overlay).
// job_type is not patchable via API; existing rows with legacy types may still be updated.
func ValidateUpdateJobInput(in CreateJobInput) error {
	return validateJobDefinitionInput(in, false)
}

func validateJobDefinitionInput(in CreateJobInput, requireImplementedJobType bool) error {
	if requireImplementedJobType {
		jt := strings.TrimSpace(in.JobType)
		if jt != "" && !IsKnowledgeJobProcessorImplemented(jt) {
			return fmt.Errorf("%w: %q (%s)", ErrUnimplementedJobType, jt, FailClosedMessageForJobType(jt))
		}
	}
	pm := strings.TrimSpace(in.ProcessingMode)
	if pm == "" {
		pm = "summarize"
	}
	if err := ValidateProcessingMode(pm); err != nil {
		return err
	}
	if err := ValidatePublicationMode(in.PublicationMode); err != nil {
		return err
	}
	norm := NormalizePublicationModeForStorage(in.PublicationMode)
	norm = NormalizePublicationMode(norm)
	if norm != PublicationModeDraftOnly && in.OutputDomainID == nil {
		return ErrOutputDomainRequired
	}
	if in.ReviewRequired && norm == PublicationModeAutoPublish {
		return ErrPublicationModeReviewConflict
	}
	if err := validateSourceScopeForJobType(in.JobType, in.SourceScopeJSON); err != nil {
		return err
	}
	if err := validateUnrestrictedScope(in.JobType, in.SourceScopeJSON); err != nil {
		return err
	}
	return nil
}

// ValidateTriggerRowsForPrimaryType ensures non-manual trigger_type has a matching active trigger row.
func ValidateTriggerRowsForPrimaryType(primaryTriggerType string, triggers []JobTriggerRow) error {
	tt := strings.TrimSpace(strings.ToLower(primaryTriggerType))
	if tt == "" || tt == "manual" {
		return nil
	}
	for i := range triggers {
		if strings.ToLower(strings.TrimSpace(triggers[i].Status)) != "active" {
			continue
		}
		if strings.TrimSpace(triggers[i].TriggerType) == tt {
			return nil
		}
	}
	return fmt.Errorf("%w for type %q", ErrTriggerRowsMismatch, primaryTriggerType)
}

func validateUnrestrictedScope(jobType string, raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	v, ok := m["unrestricted"]
	if !ok {
		return nil
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil || !b {
		return nil
	}
	// No job types allow unrestricted scope in v1.
	_ = jobType
	return ErrSourceScopeUnrestricted
}

func validateSourceScopeForJobType(jobType string, raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return ErrSourceScopeRequired
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("%w: %w", ErrSourceScopeInvalid, err)
	}
	if len(m) == 0 {
		return ErrSourceScopeRequired
	}
	switch jobType {
	case "weekly_digest":
		return validateWeeklyDigestScope(m)
	case "decision_extraction", "planning_summary", "support_trends_extraction":
		return validateSourceFeedScopedJob(m, jobType)
	case "stale_scan":
		return validateDefaultJobScope(m)
	default:
		return validateDefaultJobScope(m)
	}
}

func parseUUIDKey(m map[string]json.RawMessage, key string) (uuid.UUID, error) {
	v, ok := m[key]
	if !ok {
		return uuid.Nil, fmt.Errorf("missing %s", key)
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", key)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return uuid.Nil, fmt.Errorf("empty %s", key)
	}
	id, err := uuid.Parse(s)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fmt.Errorf("invalid %s", key)
	}
	return id, nil
}

func validateWeeklyDigestScope(m map[string]json.RawMessage) error {
	if _, err := parseUUIDKey(m, "source_feed_id"); err != nil {
		return fmt.Errorf("weekly_digest scope: %w", err)
	}
	if _, err := parseUUIDKey(m, "domain_id"); err != nil {
		return fmt.Errorf("weekly_digest scope: %w", err)
	}
	return nil
}

func validateSourceFeedScopedJob(m map[string]json.RawMessage, jobType string) error {
	if _, err := parseUUIDKey(m, "source_feed_id"); err != nil {
		return fmt.Errorf("%s scope: %w", jobType, err)
	}
	if _, err := parseUUIDKey(m, "domain_id"); err != nil {
		return fmt.Errorf("%s scope: %w", jobType, err)
	}
	return nil
}

func validateDefaultJobScope(m map[string]json.RawMessage) error {
	if _, err := parseUUIDKey(m, "domain_id"); err != nil {
		return fmt.Errorf("source scope: governed domain_id required: %w", err)
	}
	return nil
}

var (
	// ErrTriggerTypeRequired is returned when trigger_type is empty.
	ErrTriggerTypeRequired = errors.New("trigger_type required")
	// ErrTriggerScheduledMissingSchedule is returned for scheduled triggers without schedule_expr.
	ErrTriggerScheduledMissingSchedule = errors.New("scheduled trigger requires non-empty schedule_expr")
	// ErrTriggerEventFilterRequired is returned when event_driven trigger has empty event_filter_json.
	ErrTriggerEventFilterRequired = errors.New("event_driven trigger requires non-empty event_filter_json")
	// ErrTriggerWindowConfigRequired is returned when window_based trigger has empty window_config_json.
	ErrTriggerWindowConfigRequired = errors.New("window_based trigger requires non-empty window_config_json")
	// ErrTriggerConditionalJSONRequired is returned when conditional trigger JSON payloads are empty.
	ErrTriggerConditionalJSONRequired = errors.New("conditional trigger requires non-empty event_filter_json or window_config_json")
)

// ValidateTriggerInput validates trigger shape by trigger_type.
func ValidateTriggerInput(in CreateTriggerInput) error {
	if strings.TrimSpace(in.TriggerType) == "" {
		return ErrTriggerTypeRequired
	}
	switch strings.TrimSpace(in.TriggerType) {
	case "scheduled":
		if in.ScheduleExpr == nil || strings.TrimSpace(*in.ScheduleExpr) == "" {
			return ErrTriggerScheduledMissingSchedule
		}
	case "event_driven":
		if !jsonObjectNonEmpty(in.EventFilterJSON) {
			return ErrTriggerEventFilterRequired
		}
	case "window_based":
		if !jsonObjectNonEmpty(in.WindowConfigJSON) {
			return ErrTriggerWindowConfigRequired
		}
	case "conditional":
		if !jsonObjectNonEmpty(in.EventFilterJSON) && !jsonObjectNonEmpty(in.WindowConfigJSON) {
			return ErrTriggerConditionalJSONRequired
		}
	default:
		// manual and unknown types: only non-empty trigger_type required
	}
	return nil
}

// IsClientJobDefinitionError reports validation errors that should map to HTTP 400.
func IsClientJobDefinitionError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrUnknownJobTemplate) ||
		errors.Is(err, ErrJobMissingName) ||
		errors.Is(err, ErrJobMissingType) ||
		errors.Is(err, ErrUnimplementedJobType) ||
		errors.Is(err, ErrInvalidPublicationMode) ||
		errors.Is(err, ErrSourceScopeRequired) ||
		errors.Is(err, ErrSourceScopeInvalid) ||
		errors.Is(err, ErrOutputDomainRequired) ||
		errors.Is(err, ErrPublicationModeReviewConflict) ||
		errors.Is(err, ErrSourceScopeUnrestricted) ||
		errors.Is(err, ErrTriggerRowsMismatch) ||
		errors.Is(err, ErrProcessingModeInvalid)
}

// IsClientTriggerDefinitionError reports trigger validation errors for HTTP 400.
func IsClientTriggerDefinitionError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrTriggerTypeRequired) ||
		errors.Is(err, ErrTriggerScheduledMissingSchedule) ||
		errors.Is(err, ErrTriggerEventFilterRequired) ||
		errors.Is(err, ErrTriggerWindowConfigRequired) ||
		errors.Is(err, ErrTriggerConditionalJSONRequired)
}

func jsonObjectNonEmpty(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	return len(m) > 0
}
