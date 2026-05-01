# cache

L1 in-process cache (BigCache) with optional L2 Redis. Wraps `eko/gocache/v4`. Used by the v0.4.0 hot-read-path optimization for `/domains`, `/users/:id/effective-access`, `/search` keyword, and `/knowledge-jobs/engine-metadata`.

## Surface

```go
type Cache interface {
    Get(ctx, key) ([]byte, error)             // returns ErrMiss on absent
    Set(ctx, key, value, ttl) error
    Delete(ctx, key) error
    DeletePrefix(ctx, prefix) error            // namespace flush
}

func Null() Cache                               // no-op; injected when CACHE_L1_ENABLED=false
func NewBigcache(BigcacheOptions) (Cache, error) // L1 backend
type Invalidator                                  // wraps a Cache; emits typed flush events
```

## Files

- `cache.go` — `Cache` interface + `Null()` no-op backend (every Get is a miss).
- `bigcache.go` — BigCache-backed L1 with sidecar `sync.Map` for `DeletePrefix` enumeration.
- `keys.go` — typed key constructors. **Every per-user key embeds the principal UUID** to prevent cross-principal contamination. The `SafePrefixForInvalidation` helper rejects empty prefixes (defense against accidental whole-cache wipe).
- `invalidation.go` — `Invalidator` with typed methods (`EntityPublished`, `RoleGranted`, `PolicyUpdated`, `FeedConfigPatched`, `EntitySummarized`). Each maps to one or more cache prefixes.
- `cache_test.go` — 7 tests covering null behaviour, BigCache round-trip, prefix delete, principal scoping, filter-key stability, safe prefix rejection, invalidator dispatch.

## Why a `Null()` backend exists

Workers and CLI tools call `app.NewDeps`. They don't need cache. Returning `cache.Null()` instead of nil lets handlers call `Cache.Get` / `Cache.Set` unconditionally; every Get is a miss, every Set is silent. No nil-checks downstream.

## Why prefix invalidation is the primary flush primitive

Permission changes ripple. Computing the affected user set after `policy.updated` is expensive; computing it after a domain-wide entity publish is impossible without an index we don't have. The trade-off (per ADR-0014's single-instance stance): drop the whole namespace on rare events, accept the few-ms cost. TTLs are short (default 60 s) so the worst-case staleness is bounded.

## Authorization decisions are NOT cached

`AccessEvaluator.Evaluate` runs synchronously on every authorization decision. The L1 cache only stores already-decided **response bodies** (the `/domains` JSON, the `/effective-access` introspection JSON, the `/search` hits array). Operator UI affordances may be stale up to TTL; the next privileged operation enforces fresh permission checks. Documented in [PRODUCTION_HARDENING.md §12](../../../../docs/PRODUCTION_HARDENING.md).

## Configuration

Operator env vars (off by default):

- `CACHE_L1_ENABLED` — `true` to construct BigCache instead of `Null()`.
- `CACHE_L1_TTL_SECONDS` — default 60. Lower for faster permission-change propagation.
- `CACHE_L1_MAX_MB` — default 64. BigCache pre-allocates eagerly; budget alongside container memory limits.

## Adding a new cached endpoint

1. Add a typed key constructor in `keys.go` (must embed principal if per-user).
2. In the handler: read-through `d.Cache.Get(ctx, key)`; on miss, compute, `d.Cache.Set`, set `X-Cache: MISS|HIT` header for diagnosability.
3. Identify the invalidation events that should drop the new key. Either reuse an existing `Invalidator` method or add one (typed methods only — no free-form event strings).
