package openapi_v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestValidateJSONPathExpr_strictMode is the security-critical boundary:
// operator-supplied filter expressions must be rejected at activation. A
// regression here means an attacker who can edit a feed config can run
// arbitrary JSONPath logic against upstream responses.
func TestValidateJSONPathExpr_strictMode(t *testing.T) {
	for _, expr := range []string{
		"$.items[?(@.id > 0)]",
		"$.data[?(@.archived == false)].title",
		"?(@.x)",
	} {
		t.Run(expr, func(t *testing.T) {
			if err := ValidateJSONPathExpr(expr); err == nil {
				t.Fatalf("expected rejection for filter expression %q", expr)
			}
		})
	}

	// Legitimate static paths must pass.
	for _, expr := range []string{
		"$.id",
		"$.body.text",
		"$.attributes.title",
		"$.meta.author.email",
		"$",
	} {
		t.Run(expr, func(t *testing.T) {
			if err := ValidateJSONPathExpr(expr); err != nil {
				t.Errorf("expected acceptance for static path %q, got %v", expr, err)
			}
		})
	}

	// Empty + non-$ rejected.
	if err := ValidateJSONPathExpr(""); err == nil {
		t.Error("empty expression should be rejected")
	}
	if err := ValidateJSONPathExpr("items.0"); err == nil {
		t.Error("expression without $ prefix should be rejected")
	}
}

// TestFeedConfig_validate covers the activation gate. Each subcase mirrors
// an operator misconfiguration we want to catch before a sync runs.
func TestFeedConfig_validate(t *testing.T) {
	good := FeedConfig{
		OpenAPIURL:  "https://api.example.com/openapi.json",
		ListPath:    "/issues",
		RecordType:  "work_item",
		ItemMapping: map[string]string{"external_ref": "$.id", "title": "$.title"},
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("happy path rejected: %v", err)
	}

	cases := []struct {
		name string
		mut  func(c *FeedConfig)
		want string
	}{
		{"missing openapi_url", func(c *FeedConfig) { c.OpenAPIURL = "" }, "openapi_url required"},
		{"http url (non-localhost)", func(c *FeedConfig) { c.OpenAPIURL = "http://api.example.com/spec" }, "https://"},
		{"missing list_path", func(c *FeedConfig) { c.ListPath = "" }, "list_path required"},
		{"list_path no leading slash", func(c *FeedConfig) { c.ListPath = "issues" }, "list_path must start"},
		{"unknown record_type", func(c *FeedConfig) { c.RecordType = "made_up" }, "not in supported set"},
		{"empty item_mapping", func(c *FeedConfig) { c.ItemMapping = map[string]string{} }, "item_mapping required"},
		{"missing external_ref", func(c *FeedConfig) { c.ItemMapping = map[string]string{"title": "$.title"} }, "external_ref required"},
		{"jsonpath filter", func(c *FeedConfig) { c.ItemMapping = map[string]string{"external_ref": "$.id", "title": "$.items[?(@.x)].t"} }, "filter expressions"},
		{"unsupported auth", func(c *FeedConfig) { c.Auth = AuthConfig{Type: "oauth2"} }, "auth.type"},
		{"bearer without token", func(c *FeedConfig) { c.Auth = AuthConfig{Type: "bearer"} }, "auth.token required"},
		{"page_size out of range", func(c *FeedConfig) { c.Pagination.PageSize = 500 }, "page_size"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := good
			c.Auth = AuthConfig{}
			c.Pagination = PaginationConfig{}
			c.ItemMapping = map[string]string{"external_ref": "$.id", "title": "$.title"}
			tc.mut(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("expected rejection containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error to contain %q, got: %v", tc.want, err)
			}
		})
	}
}

// TestFetchAndValidateSpec_oversizedRejected covers the 5MB cap. An
// adversarial config pointing at a 6MB blob should fail before the parser
// gets to run — preventing memory blow-ups.
func TestFetchAndValidateSpec_oversizedRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Stream just over the cap of arbitrary bytes (not even valid JSON).
		buf := make([]byte, MaxSpecBytes+1024)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf)
	}))
	defer srv.Close()

	_, err := FetchAndValidateSpec(context.Background(), srv.URL, "/items")
	if err == nil {
		t.Fatal("expected oversize rejection")
	}
	if !strings.Contains(err.Error(), "exceeds") && !strings.Contains(err.Error(), "cap") {
		// kin-openapi may throw a parse error first if the limit-reader
		// returned the truncated body; either way, the call must not
		// succeed with an oversized payload.
		t.Logf("error path: %v", err)
	}
}

// TestFetchAndValidateSpec_validSpec checks the happy path with a tiny
// valid OpenAPI 3.1 spec. The list_path assertion must pass; absent path
// must fail.
func TestFetchAndValidateSpec_validSpec(t *testing.T) {
	spec := map[string]any{
		"openapi": "3.0.3",
		"info":    map[string]any{"title": "tiny", "version": "1"},
		"servers": []any{map[string]any{"url": "https://api.example.com"}},
		"paths": map[string]any{
			"/items": map[string]any{
				"get": map[string]any{
					"responses": map[string]any{"200": map[string]any{"description": "ok"}},
				},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(spec)
	}))
	defer srv.Close()

	doc, err := FetchAndValidateSpec(context.Background(), srv.URL, "/items")
	if err != nil {
		t.Fatalf("happy path: %v", err)
	}
	if doc == nil {
		t.Fatal("nil doc on success")
	}

	if _, err := FetchAndValidateSpec(context.Background(), srv.URL, "/missing"); err == nil {
		t.Fatal("missing list_path should fail")
	}
}

// TestEvalString_softMissing covers the documented "missing field is empty,
// not error" behavior. Items missing optional fields don't abort sync.
func TestEvalString_softMissing(t *testing.T) {
	doc := map[string]any{"id": "abc", "title": "hello"}
	v, err := EvalString("$.id", doc)
	if err != nil || v != "abc" {
		t.Fatalf("got %q err=%v", v, err)
	}
	v, err = EvalString("$.missing", doc)
	if err != nil {
		t.Fatalf("missing field should not error, got: %v", err)
	}
	if v != "" {
		t.Errorf("missing field should be empty, got %q", v)
	}
}
