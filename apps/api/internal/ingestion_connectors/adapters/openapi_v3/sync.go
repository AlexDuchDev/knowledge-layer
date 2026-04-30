package openapi_v3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PageItem is one canonical-shape record extracted from the upstream
// response after JSONPath mapping. Used by the connector orchestrator to
// build raw_artifacts + normalized_records.
type PageItem struct {
	ExternalRef     string
	NormalizedJSON  json.RawMessage // the structured payload to persist
	RawJSON         json.RawMessage // verbatim upstream item for raw_artifact
	RecordHash      string          // sha256 of NormalizedJSON for dedup
	RawHash         string          // sha256 of RawJSON for raw_artifact dedup
}

// SyncResult is what one sync run reports back to the orchestrator. The
// connector framework's per-feed metrics use these counts.
type SyncResult struct {
	PagesFetched int
	ItemsSeen    int
	ItemsValid   int // items that produced a non-empty external_ref
	Truncated    bool
}

// Run executes a single sync against an openapi_v3 feed config. The caller
// invokes onItem for each parsed PageItem; persistence (raw_artifact +
// normalized_record + chunk_rebuild trigger) lives one layer up so this
// package stays decoupled from the rest of ingestion_connectors.
//
// Run honors:
//   - PageSize / MaxPages bounds.
//   - Pagination via offset/limit query params (configurable names).
//   - Bearer auth (only auth scheme in v0.6.0).
//   - 30 s per-request timeout.
//   - Truncated short-circuit when fewer items returned than PageSize
//     (last page).
func Run(ctx context.Context, baseURL string, cfg *FeedConfig, httpClient *http.Client, onItem func(PageItem) error) (*SyncResult, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	pageSize, maxPages := cfg.Pagination.Effective()
	offsetParam := cfg.Pagination.EffectiveOffsetParam()
	limitParam := cfg.Pagination.EffectiveLimitParam()
	listPointer := strings.TrimSpace(cfg.ListPointer)
	if listPointer == "" {
		listPointer = "$" // assume the response IS the array
	}

	res := &SyncResult{}
	endpoint, err := url.JoinPath(baseURL, cfg.ListPath)
	if err != nil {
		return res, fmt.Errorf("openapi_v3: join base + list_path: %w", err)
	}

	for page := 0; page < maxPages; page++ {
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		default:
		}

		offset := page * pageSize
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return res, err
		}
		q := req.URL.Query()
		q.Set(offsetParam, strconv.Itoa(offset))
		q.Set(limitParam, strconv.Itoa(pageSize))
		req.URL.RawQuery = q.Encode()
		req.Header.Set("Accept", "application/json")
		if cfg.Auth.Type == "bearer" && cfg.Auth.Token != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.Auth.Token)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return res, fmt.Errorf("openapi_v3: page %d: %w", page, err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return res, fmt.Errorf("openapi_v3: read page %d: %w", page, readErr)
		}
		if resp.StatusCode/100 != 2 {
			return res, fmt.Errorf("openapi_v3: page %d returned %d: %s", page, resp.StatusCode, snippet(body))
		}
		res.PagesFetched++

		var doc any
		if err := json.Unmarshal(body, &doc); err != nil {
			return res, fmt.Errorf("openapi_v3: parse page %d: %w", page, err)
		}
		items, err := EvalArray(listPointer, doc)
		if err != nil {
			return res, fmt.Errorf("openapi_v3: extract items page %d: %w", page, err)
		}

		for _, raw := range items {
			res.ItemsSeen++
			pi, err := mapItem(cfg, raw)
			if err != nil {
				// Skip individual bad items — mapping errors usually mean
				// one bad row in upstream, not a config problem.
				continue
			}
			if pi.ExternalRef == "" {
				continue
			}
			res.ItemsValid++
			if err := onItem(pi); err != nil {
				return res, fmt.Errorf("openapi_v3: persist item %s: %w", pi.ExternalRef, err)
			}
		}

		if len(items) < pageSize {
			// Last page; short-circuit instead of scanning to maxPages.
			break
		}
		if page+1 == maxPages {
			res.Truncated = true
		}
	}
	return res, nil
}

// mapItem applies cfg.ItemMapping to a single response item. Builds:
//   - the normalized payload (canonical_field → string value) for
//     normalized_records.structured_payload_json,
//   - the raw payload (verbatim item) for raw_artifacts.payload_bytes,
//   - SHA-256 hashes for dedup.
func mapItem(cfg *FeedConfig, raw any) (PageItem, error) {
	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return PageItem{}, err
	}
	rawSum := sha256.Sum256(rawBytes)

	normalized := map[string]string{
		"connector_family": "openapi_v3",
		"connector_type":   "openapi_v3",
		"record_type":      cfg.RecordType,
	}
	for canonicalField, expr := range cfg.ItemMapping {
		v, _ := EvalString(expr, raw)
		normalized[canonicalField] = v
	}
	normBytes, err := json.Marshal(normalized)
	if err != nil {
		return PageItem{}, err
	}
	normSum := sha256.Sum256(normBytes)

	return PageItem{
		ExternalRef:    normalized["external_ref"],
		NormalizedJSON: normBytes,
		RawJSON:        rawBytes,
		RecordHash:     hex.EncodeToString(normSum[:]),
		RawHash:        hex.EncodeToString(rawSum[:]),
	}, nil
}

// snippet truncates an upstream error body so the operator-visible error
// message stays readable in logs / audit metadata.
func snippet(b []byte) string {
	const max = 240
	s := bytes.TrimSpace(b)
	if len(s) > max {
		return string(s[:max]) + "…"
	}
	return string(s)
}
