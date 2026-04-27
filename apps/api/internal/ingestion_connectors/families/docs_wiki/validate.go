package docs_wiki

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// NotionFeedConfigJSON is validated keys for Notion feeds (non-secret token checked elsewhere).
type NotionFeedConfigJSON struct {
	Scope string `json:"scope,omitempty"` // "page" | "database"
}

// ParseNotionConfig parses connector_config_json for Notion (scope; token parsed separately).
func ParseNotionConfig(raw json.RawMessage) (NotionFeedConfigJSON, error) {
	var c NotionFeedConfigJSON
	if len(raw) == 0 {
		return c, nil
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return NotionFeedConfigJSON{}, err
	}
	return c, nil
}

// ValidateNotionScope returns an error if scope is set and invalid.
func ValidateNotionScope(scope string) error {
	s := strings.TrimSpace(strings.ToLower(scope))
	if s == "" {
		return errors.New("notion: connector_config_json.scope required (page or database)")
	}
	switch s {
	case "page", "database":
		return nil
	default:
		return fmt.Errorf("notion: invalid scope %q", scope)
	}
}

// RequireNotionToken checks integration token presence (stored in connector_config_json.notion_integration_token for v1).
func RequireNotionToken(raw json.RawMessage) (string, error) {
	var m struct {
		NotionIntegrationToken string `json:"notion_integration_token"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", err
	}
	t := strings.TrimSpace(m.NotionIntegrationToken)
	if t == "" {
		return "", errors.New("notion: notion_integration_token required in connector_config_json")
	}
	return t, nil
}
