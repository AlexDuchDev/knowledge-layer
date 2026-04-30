package cache

import (
	"context"
	"log"

	"github.com/google/uuid"
)

// Invalidator drops cached entries when state-changing events happen. Every
// public method maps to one or more events emitted from the audit/event bus
// (entity.published, role.granted, feed.config_patched, policy.updated,
// entity.summarized). Methods are deliberately specific instead of taking a
// free-form event type so a misnamed event can't silently pass.
//
// Concrete invalidation rules:
//
//   - entity.published / entity.summarized → drop NSSearchKeyword (any
//     keyword search response could now rank that entity differently). We
//     drop all of NSSearchKeyword instead of computing affected principals;
//     that's acceptable because keyword search responses have a TTL of
//     ~30s anyway.
//
//   - role.granted (or revoked, or sensitivity-cap change) → drop NSDomains
//     and NSEffectiveAccess scoped to the affected user. Both rely on the
//     access graph; a stale read could mislead operator UI for up to TTL.
//
//   - policy.updated → drop ALL NSEffectiveAccess. Policy changes can ripple
//     across users; computing the affected set is expensive and rare so we
//     flush the namespace.
//
//   - feed.config_patched → drop NSSearchKeyword (feed config affects what
//     records ingest, indirectly affecting search results).
type Invalidator struct {
	cache Cache
}

// NewInvalidator wires a cache into the invalidation surface. Pass Null() to
// disable invalidation calls entirely (useful in tests where the cache is a
// no-op).
func NewInvalidator(c Cache) *Invalidator {
	if c == nil {
		c = Null()
	}
	return &Invalidator{cache: c}
}

// EntityPublished drops keyword-search caches because the new entity (or
// updated lifecycle state) may now appear in result rankings.
func (i *Invalidator) EntityPublished(ctx context.Context, entityID uuid.UUID) {
	i.deletePrefix(ctx, NSSearchKeyword)
}

// EntitySummarized fires after an entity_summarize job writes
// synthesized_summary. OpenSearch reindex includes it in the indexed text,
// so /search ranking can shift.
func (i *Invalidator) EntitySummarized(ctx context.Context, entityID uuid.UUID) {
	i.deletePrefix(ctx, NSSearchKeyword)
}

// RoleGranted drops the affected user's NSDomains and NSEffectiveAccess
// entries. The "affected user" is the role grantee; the granter's caches
// are unchanged.
func (i *Invalidator) RoleGranted(ctx context.Context, grantee uuid.UUID) {
	i.deletePrefix(ctx, PrefixForPrincipal(NSDomains, grantee))
	i.deletePrefix(ctx, PrefixForPrincipal(NSEffectiveAccess, grantee))
}

// PolicyUpdated drops the entire NSEffectiveAccess namespace because policy
// changes can ripple across any principal. Policies change rarely; this is
// the operator-friendly trade-off.
func (i *Invalidator) PolicyUpdated(ctx context.Context) {
	i.deletePrefix(ctx, NSEffectiveAccess)
	i.deletePrefix(ctx, NSDomains)
}

// FeedConfigPatched drops keyword-search caches; feed config affects what is
// ingested and ranked. Future scope: also drop search results that referenced
// records from that feed (more surgical), but for now namespace flush is OK.
func (i *Invalidator) FeedConfigPatched(ctx context.Context, feedID uuid.UUID) {
	i.deletePrefix(ctx, NSSearchKeyword)
}

func (i *Invalidator) deletePrefix(ctx context.Context, prefix string) {
	safe, ok := SafePrefixForInvalidation(prefix)
	if !ok {
		// An empty / whitespace prefix would wipe the entire cache.
		// Refuse loudly; this is always a coding bug.
		log.Printf("cache: refusing empty prefix invalidation; check call site")
		return
	}
	if err := i.cache.DeletePrefix(ctx, safe); err != nil {
		// Best-effort: log and move on. Failing the originating audit
		// event because of a cache miss-clear would be worse than a
		// stale read for one TTL window.
		log.Printf("cache: invalidation %q failed: %v", safe, err)
	}
}
