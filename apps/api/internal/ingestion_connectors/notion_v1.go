package ingestion_connectors

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
)

const notionAPIVersion = "2022-06-28"

// ValidateNotionV1ForActivation checks token, scope, and external_ref.
func ValidateNotionV1ForActivation(feed *SourceFeed, token string, scopeCfg docs_wiki.NotionFeedConfigJSON) error {
	if feed == nil {
		return errors.New("notion: missing feed")
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("notion: integration token required")
	}
	if err := docs_wiki.ValidateNotionScope(scopeCfg.Scope); err != nil {
		return err
	}
	if strings.TrimSpace(feed.ExternalRef) == "" {
		return errors.New("notion: external_ref required (page or database id)")
	}
	return nil
}

// notionPageProperties holds page retrieve response fragments we need.
type notionPageProperties struct {
	Properties map[string]json.RawMessage `json:"properties"`
}

func notionExtractTitle(props notionPageProperties) string {
	for _, raw := range props.Properties {
		var wrap struct {
			Type  string          `json:"type"`
			Title json.RawMessage `json:"title"`
		}
		if err := json.Unmarshal(raw, &wrap); err != nil || wrap.Type != "title" {
			continue
		}
		var chunks []struct {
			PlainText string `json:"plain_text"`
		}
		if err := json.Unmarshal(wrap.Title, &chunks); err != nil {
			continue
		}
		var b strings.Builder
		for _, c := range chunks {
			b.WriteString(c.PlainText)
		}
		if s := strings.TrimSpace(b.String()); s != "" {
			return s
		}
	}
	return ""
}

type notionListBlocksResp struct {
	Results    []json.RawMessage `json:"results"`
	NextCursor string            `json:"next_cursor"`
	HasMore    bool              `json:"has_more"`
}

type notionDatabaseQueryResp struct {
	Results []struct {
		ID string `json:"id"`
	} `json:"results"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor"`
}

func notionBlockPlainLines(blockJSON []byte) []string {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(blockJSON, &probe); err != nil {
		return nil
	}
	// Rich-text blocks share "rich_text" arrays on type-specific keys.
	type richWrap struct {
		RichText []struct {
			PlainText string `json:"plain_text"`
		} `json:"rich_text"`
	}
	paths := []string{
		probe.Type,
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(blockJSON, &m)
	var lines []string
	for _, p := range paths {
		if p == "" || m == nil {
			continue
		}
		raw, ok := m[p]
		if !ok {
			continue
		}
		var rw richWrap
		if err := json.Unmarshal(raw, &rw); err != nil {
			continue
		}
		var b strings.Builder
		for _, rt := range rw.RichText {
			b.WriteString(rt.PlainText)
		}
		if s := strings.TrimSpace(b.String()); s != "" {
			lines = append(lines, s)
		}
	}
	return lines
}
