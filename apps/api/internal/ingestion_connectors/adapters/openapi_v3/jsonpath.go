package openapi_v3

import (
	"errors"
	"fmt"
	"strings"

	"github.com/PaesslerAG/jsonpath"
)

// ValidateJSONPathExpr is the activation-time gate that prevents operator-
// supplied JSONPath from doing more than read a value. The PaesslerAG library
// supports filter expressions (`?(@.x > 0)`) which open a small expression-
// language attack surface — adversarial config could use them to probe the
// upstream response shape or trigger unintended evaluation. We defend by
// rejecting any `?(` substring at validation; legitimate operator JSONPaths
// for KL connector mapping never need filter expressions (they always pluck
// known fields from a known response).
//
// Defense rationale per ADR-0016: the connector's job is "extract field X
// from each item"; that's `$.field` or `$.nested.field` — both static.
// Anything more complex is a config smell, almost certainly a bug, and the
// rejection forces a clearer feed config.
func ValidateJSONPathExpr(expr string) error {
	if strings.TrimSpace(expr) == "" {
		return errors.New("empty JSONPath expression")
	}
	if !strings.HasPrefix(expr, "$") {
		return fmt.Errorf("JSONPath must start with '$' (got %q)", expr)
	}
	// The exact attack vector — filter scripts.
	if strings.Contains(expr, "?(") {
		return fmt.Errorf("filter expressions ?(...) are not allowed in v0.6.0 (got %q); use a static field path", expr)
	}
	// Compile to surface syntactic errors at activation rather than first
	// sync. The compiled object is discarded; we just need the parse-time
	// validation.
	if _, err := jsonpath.New(expr); err != nil {
		return fmt.Errorf("JSONPath syntax: %w", err)
	}
	return nil
}

// EvalString executes a JSONPath against a parsed JSON document and returns
// the first matched value as a string. Used to map response items into
// normalized_record fields. Returns "" + nil when the expression matches
// nothing — operator-friendly default is "missing field", not error.
func EvalString(expr string, doc any) (string, error) {
	res, err := jsonpath.Get(expr, doc)
	if err != nil {
		// jsonpath returns errors for "no match"; treat as soft empty so
		// individual records with missing optional fields don't abort
		// the whole sync.
		return "", nil
	}
	return stringify(res), nil
}

// EvalArray returns the first matched value as a []any, used for the
// list_pointer (path to the items array inside the response).
func EvalArray(expr string, doc any) ([]any, error) {
	res, err := jsonpath.Get(expr, doc)
	if err != nil {
		return nil, fmt.Errorf("jsonpath %q: %w", expr, err)
	}
	arr, ok := res.([]any)
	if !ok {
		return nil, fmt.Errorf("jsonpath %q matched non-array (%T)", expr, res)
	}
	return arr, nil
}

// stringify renders a json-decoded value as a string. Strings pass through;
// numbers / bools render via fmt.Sprint; nested maps/arrays render as JSON
// for visibility (rare; an operator who maps `external_ref` to a struct is
// already misconfiguring the feed, but we don't want to silently lose data).
func stringify(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%v", t)
	case bool:
		return fmt.Sprintf("%v", t)
	default:
		return fmt.Sprintf("%v", v)
	}
}
