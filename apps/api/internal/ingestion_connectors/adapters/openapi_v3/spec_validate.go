package openapi_v3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// MaxSpecBytes caps the OpenAPI spec download. 5 MB is generous for any
// real REST API doc; oversized specs are almost always an attacker pointing
// at a non-spec URL or a misconfigured server.
const MaxSpecBytes = 5 * 1024 * 1024

// FetchAndValidateSpec downloads the OpenAPI document at openAPIURL,
// validates it, and asserts the operator-specified ListPath exists as a GET
// operation. Defense in depth:
//
//   - Body cap (MaxSpecBytes) prevents memory blow-ups from giant responses.
//   - 10 s timeout prevents indefinite hangs on slow upstreams.
//   - kin-openapi's loader catches syntactic errors + circular $refs.
//   - Manual ListPath assertion ensures the operator's pagination knob
//     actually targets a documented operation; otherwise the first sync
//     would fail in a confusing way.
func FetchAndValidateSpec(ctx context.Context, openAPIURL, listPath string) (*openapi3.T, error) {
	dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(dctx, http.MethodGet, openAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("openapi_v3: build spec request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openapi_v3: fetch spec: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("openapi_v3: spec fetch returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxSpecBytes+1))
	if err != nil {
		return nil, fmt.Errorf("openapi_v3: read spec: %w", err)
	}
	if len(body) > MaxSpecBytes {
		return nil, fmt.Errorf("openapi_v3: spec exceeds %d-byte cap", MaxSpecBytes)
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false // local refs only — external $refs are an SSRF vector
	doc, err := loader.LoadFromData(body)
	if err != nil {
		return nil, fmt.Errorf("openapi_v3: parse spec: %w", err)
	}
	if err := doc.Validate(dctx); err != nil {
		return nil, fmt.Errorf("openapi_v3: validate spec: %w", err)
	}

	pathItem := doc.Paths.Find(listPath)
	if pathItem == nil {
		return nil, fmt.Errorf("openapi_v3: list_path %q not present in spec", listPath)
	}
	if pathItem.Get == nil {
		return nil, fmt.Errorf("openapi_v3: list_path %q has no GET operation", listPath)
	}
	return doc, nil
}

// ResolveBaseURL picks the base URL for runtime requests: operator override
// > spec's first server URL > error. The path-prefix in the spec server URL
// is preserved so list_path + base behave consistently.
func ResolveBaseURL(doc *openapi3.T, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if doc == nil || len(doc.Servers) == 0 || doc.Servers[0].URL == "" {
		return "", errors.New("openapi_v3: spec has no servers and no base_url override")
	}
	return doc.Servers[0].URL, nil
}
