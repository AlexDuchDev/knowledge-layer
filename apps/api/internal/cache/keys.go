package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// Key constructors live here so that no caller is tempted to build a cache
// key inline and skip the principal-scoping rule. Every per-user response
// MUST include the principal in its key — otherwise alice can read bob's
// /effective-access response if bob asked first.
//
// The naming convention is `<namespace>:<principal>:<discriminator>`. The
// namespace prefix is what the invalidator targets via DeletePrefix; for
// example, dropping NSDomains drops every per-user /domains entry in one
// call. Discriminators are hashed when they could include arbitrary user
// input (search queries, filter dictionaries) so cache keys remain bounded.

const (
	NSDomains          = "domains:"
	NSEffectiveAccess  = "eff-access:"
	NSSearchKeyword    = "search-kw:"
	NSEngineMetadata   = "engine-meta:"
	NSEntitySummarized = "entity-sum:"
)

// DomainsKey scopes the /domains response to the requesting principal so a
// later read by a different user does not return a leaked list.
func DomainsKey(principal uuid.UUID) string {
	return NSDomains + principal.String()
}

// EffectiveAccessKey carries both the principal asking AND the target user.
// /users/:id/effective-access can be requested by an admin against another
// user; both axes must be in the key.
func EffectiveAccessKey(principal, target uuid.UUID) string {
	return NSEffectiveAccess + principal.String() + ":" + target.String()
}

// SearchKeywordKey hashes the (query, filters) payload into a stable suffix.
// The hash collapses semantically equivalent filter dictionaries (different
// map ordering) and stops cache keys from growing unbounded with arbitrary
// query text. Principal stays in plaintext for invalidation by user.
func SearchKeywordKey(principal uuid.UUID, query string, filters map[string]string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(query))
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = h.Write([]byte("\x00"))
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte("="))
		_, _ = h.Write([]byte(filters[k]))
	}
	return NSSearchKeyword + principal.String() + ":" + hex.EncodeToString(h.Sum(nil)[:12])
}

// EngineMetadataKey is global — implemented job_types do not vary per user.
// Invalidated only on process restart (which is fine: the list changes only
// when a new release ships).
func EngineMetadataKey() string {
	return NSEngineMetadata + "global"
}

// PrefixForPrincipal returns the cache prefix for entries belonging to a
// specific principal — used when invalidating after a role grant or
// effective-access change for that one user.
func PrefixForPrincipal(ns string, principal uuid.UUID) string {
	return ns + principal.String() + ":"
}

// SafePrefixForInvalidation defends against accidental empty-prefix
// invalidation that would wipe the entire cache. An empty / whitespace-only
// prefix is rejected.
func SafePrefixForInvalidation(prefix string) (string, bool) {
	p := strings.TrimSpace(prefix)
	if p == "" {
		return "", false
	}
	return p, true
}
