// Package openapi_v3 implements a generic HTTP connector that polls an
// arbitrary REST endpoint described by an OpenAPI 3.0/3.1 spec and maps
// each response item into a Knowledge Layer normalized_record via
// operator-supplied JSONPath expressions. Replaces per-vendor sync files
// for simple paginated REST APIs.
//
// v0.6.0 scope (per ADR-0016):
//
//   - Pagination: offset/limit only. Cursor and link-header → v0.7+.
//   - JSONPath: strict mode — no scripting (?(...) filters rejected at
//     activation). Defends against operator-supplied expressions executing
//     unsafe lookups.
//   - record_type: closed enum mapped to chunks/extract.go's 14 known
//     types. New types come from extending that registry, not from
//     connector config.
//   - Spec size cap: 5 MB. Bigger specs get a clear "too large" error
//     at activation, preventing memory blow-ups from operator-supplied
//     URLs.
//   - Auth: bearer token only in v0.6.0. OAuth2 client_credentials and
//     other schemes → follow-up.
package openapi_v3

import (
	"errors"
	"fmt"
	"strings"
)

// FeedConfig is what an operator supplies in source_feeds.connector_config_json
// to drive an openapi_v3 sync. JSON-tagged so the existing per-feed config
// store works as-is.
type FeedConfig struct {
	OpenAPIURL  string                  `json:"openapi_url"`
	BaseURL     string                  `json:"base_url,omitempty"` // overrides spec server URL when set
	ListPath    string                  `json:"list_path"`          // path to the listing operation, e.g. "/issues"
	Auth        AuthConfig              `json:"auth,omitempty"`
	Pagination  PaginationConfig        `json:"pagination,omitempty"`
	RecordType  string                  `json:"record_type"` // one of chunks/extract.go's 14 known types
	ItemMapping map[string]string       `json:"item_mapping"` // canonical_field → JSONPath into a single response item
	ListPointer string                  `json:"list_pointer,omitempty"` // JSONPath to the array of items inside the response (default: "$"; use "$.data" for envelope-shaped responses)
}

// AuthConfig is bearer-only in v0.6.0. Future iterations add api_key
// header, oauth2 client_credentials, etc.
type AuthConfig struct {
	Type  string `json:"type"`            // "" (none) | "bearer"
	Token string `json:"token,omitempty"` // sensitive — must be encrypted at rest like other connector secrets
}

// PaginationConfig captures offset/limit query parameter names. Defaults to
// "offset"/"limit" if both empty. v0.6.0 is offset/limit-only by design.
type PaginationConfig struct {
	OffsetParam string `json:"offset_param,omitempty"`
	LimitParam  string `json:"limit_param,omitempty"`
	PageSize    int    `json:"page_size,omitempty"`  // default 50, hard cap 200
	MaxPages    int    `json:"max_pages,omitempty"`  // default 20, hard cap 100 — guard against runaway syncs
}

// SupportedRecordTypes is the closed enum the openapi_v3 adapter accepts.
// Mirrors the 14 cases in chunks/extract.go. Validated at activation time;
// a typo from the operator gets a clear error rather than a silent no-op.
var SupportedRecordTypes = map[string]struct{}{
	"chat_message":          {},
	"chat_thread":           {},
	"docs_page":             {},
	"meeting_transcript":    {},
	"calendar_event":        {},
	"email_message":         {},
	"work_item":             {},
	"support_ticket":        {},
	"support_conversation":  {},
	"crm_record":            {},
	"google_drive_document": {},
	"m365_mail_message":     {},
	"m365_teams_message":    {},
	"m365_cloud_file":       {},
	"m365_calendar_event":   {},
}

// Validate enforces the operator-supplied config invariants. Called at
// source-feed activation (Service.ValidateSourceFeed) so a bad config never
// becomes an active feed.
func (c *FeedConfig) Validate() error {
	if strings.TrimSpace(c.OpenAPIURL) == "" {
		return errors.New("openapi_v3: openapi_url required")
	}
	if !strings.HasPrefix(c.OpenAPIURL, "https://") && !strings.HasPrefix(c.OpenAPIURL, "http://localhost") {
		return fmt.Errorf("openapi_v3: openapi_url must be https:// (or http://localhost for local testing); got %q", c.OpenAPIURL)
	}
	if strings.TrimSpace(c.ListPath) == "" {
		return errors.New("openapi_v3: list_path required")
	}
	if !strings.HasPrefix(c.ListPath, "/") {
		return fmt.Errorf("openapi_v3: list_path must start with '/'; got %q", c.ListPath)
	}
	if _, ok := SupportedRecordTypes[c.RecordType]; !ok {
		return fmt.Errorf("openapi_v3: record_type %q not in supported set (one of: chat_message, docs_page, work_item, ...). The set is closed; new types extend chunks/extract.go", c.RecordType)
	}
	if len(c.ItemMapping) == 0 {
		return errors.New("openapi_v3: item_mapping required (need at least 'external_ref' and one text field)")
	}
	if _, ok := c.ItemMapping["external_ref"]; !ok {
		return errors.New("openapi_v3: item_mapping.external_ref required (used to dedupe across syncs)")
	}
	for field, expr := range c.ItemMapping {
		if err := ValidateJSONPathExpr(expr); err != nil {
			return fmt.Errorf("openapi_v3: item_mapping.%s: %w", field, err)
		}
	}
	if c.ListPointer != "" {
		if err := ValidateJSONPathExpr(c.ListPointer); err != nil {
			return fmt.Errorf("openapi_v3: list_pointer: %w", err)
		}
	}
	if c.Auth.Type != "" && c.Auth.Type != "bearer" {
		return fmt.Errorf("openapi_v3: auth.type %q not supported in v0.6.0 (use \"bearer\" or omit)", c.Auth.Type)
	}
	if c.Auth.Type == "bearer" && strings.TrimSpace(c.Auth.Token) == "" {
		return errors.New("openapi_v3: auth.token required when auth.type=bearer")
	}
	if c.Pagination.PageSize < 0 || c.Pagination.PageSize > 200 {
		return fmt.Errorf("openapi_v3: pagination.page_size must be 1..200 (or 0 for default 50); got %d", c.Pagination.PageSize)
	}
	if c.Pagination.MaxPages < 0 || c.Pagination.MaxPages > 100 {
		return fmt.Errorf("openapi_v3: pagination.max_pages must be 1..100 (or 0 for default 20); got %d", c.Pagination.MaxPages)
	}
	return nil
}

// PageSize returns the operator's page_size or the default 50.
func (p PaginationConfig) Effective() (int, int) {
	pageSize := p.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	maxPages := p.MaxPages
	if maxPages <= 0 {
		maxPages = 20
	}
	return pageSize, maxPages
}

// EffectiveOffsetParam returns the operator override or "offset".
func (p PaginationConfig) EffectiveOffsetParam() string {
	if v := strings.TrimSpace(p.OffsetParam); v != "" {
		return v
	}
	return "offset"
}

// EffectiveLimitParam returns the operator override or "limit".
func (p PaginationConfig) EffectiveLimitParam() string {
	if v := strings.TrimSpace(p.LimitParam); v != "" {
		return v
	}
	return "limit"
}
