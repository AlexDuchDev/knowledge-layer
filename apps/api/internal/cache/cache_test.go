package cache

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestNullCache_neverStores asserts the no-op cache behaves as documented:
// every Get is a miss, every Set is silent. Hot-path handlers inject this
// when caching is disabled and rely on Get always returning ErrMiss.
func TestNullCache_neverStores(t *testing.T) {
	ctx := context.Background()
	c := Null()
	if err := c.Set(ctx, "foo", []byte("x"), time.Minute); err != nil {
		t.Fatalf("null Set: %v", err)
	}
	v, err := c.Get(ctx, "foo")
	if err != ErrMiss {
		t.Fatalf("expected ErrMiss, got value=%q err=%v", v, err)
	}
}

// TestBigcache_setGetDelete is the simple round-trip: write a key, read it
// back, delete it, observe the miss. Catches signature regressions and
// verifies the Delete/keys-mirror sync behaviour.
func TestBigcache_setGetDelete(t *testing.T) {
	c, err := NewBigcache(BigcacheOptions{DefaultTTL: time.Minute, MaxMB: 16})
	if err != nil {
		t.Fatalf("NewBigcache: %v", err)
	}
	ctx := context.Background()

	if err := c.Set(ctx, "k1", []byte("hello"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	v, err := c.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(v) != "hello" {
		t.Fatalf("round-trip mismatch: %q", v)
	}
	if err := c.Delete(ctx, "k1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.Get(ctx, "k1"); err != ErrMiss {
		t.Fatalf("expected miss post-Delete, got %v", err)
	}
}

// TestBigcache_deletePrefix asserts the namespace flush behaviour the
// invalidator depends on. After DeletePrefix("foo:") only foo:* keys are
// removed; sibling namespaces stay intact.
func TestBigcache_deletePrefix(t *testing.T) {
	c, err := NewBigcache(BigcacheOptions{DefaultTTL: time.Minute, MaxMB: 16})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	keys := []string{"foo:a", "foo:b", "bar:c", "foo:d"}
	for _, k := range keys {
		_ = c.Set(ctx, k, []byte("v"), time.Minute)
	}
	if err := c.DeletePrefix(ctx, "foo:"); err != nil {
		t.Fatalf("DeletePrefix: %v", err)
	}
	for _, k := range []string{"foo:a", "foo:b", "foo:d"} {
		if _, err := c.Get(ctx, k); err != ErrMiss {
			t.Errorf("expected miss for %q, got %v", k, err)
		}
	}
	if _, err := c.Get(ctx, "bar:c"); err != nil {
		t.Errorf("bar:c should survive, got %v", err)
	}
}

// TestKeys_principalScoping is the safety-critical test: cache keys for
// per-user resources MUST embed the principal. A regression here means
// alice could read bob's /effective-access response.
func TestKeys_principalScoping(t *testing.T) {
	alice := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	bob := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	if DomainsKey(alice) == DomainsKey(bob) {
		t.Fatal("domains key MUST differ across principals")
	}
	if EffectiveAccessKey(alice, alice) == EffectiveAccessKey(bob, alice) {
		t.Fatal("effective-access key MUST encode the asking principal")
	}
	if EffectiveAccessKey(alice, alice) == EffectiveAccessKey(alice, bob) {
		t.Fatal("effective-access key MUST encode the target principal")
	}
	if SearchKeywordKey(alice, "q", nil) == SearchKeywordKey(bob, "q", nil) {
		t.Fatal("search-keyword key MUST differ across principals")
	}
	// Engine metadata is global on purpose — check it doesn't accidentally
	// embed a principal.
	if !strings.Contains(EngineMetadataKey(), "global") {
		t.Errorf("engine-metadata key should be global, got %q", EngineMetadataKey())
	}
}

// TestKeys_filterStable asserts that map-iteration ordering does NOT change
// the cache key for the same logical filter set. Without sorted iteration,
// two equivalent requests would create two cache entries.
func TestKeys_filterStable(t *testing.T) {
	p := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	a := SearchKeywordKey(p, "q", map[string]string{"x": "1", "y": "2", "z": "3"})
	b := SearchKeywordKey(p, "q", map[string]string{"z": "3", "x": "1", "y": "2"})
	if a != b {
		t.Errorf("filter-key not stable across map ordering: %s vs %s", a, b)
	}
}

// TestSafePrefixForInvalidation rejects empty prefixes that would wipe the
// entire cache.
func TestSafePrefixForInvalidation(t *testing.T) {
	for _, in := range []string{"", "   ", "\t"} {
		if _, ok := SafePrefixForInvalidation(in); ok {
			t.Errorf("empty/whitespace prefix %q should be rejected", in)
		}
	}
	if got, ok := SafePrefixForInvalidation("foo:"); !ok || got != "foo:" {
		t.Errorf("legit prefix should pass: got=%q ok=%v", got, ok)
	}
}

// TestInvalidator_dispatch checks that role/policy/entity invalidation calls
// reach the cache layer. Uses a stub Cache to record which prefixes were
// asked to be deleted.
func TestInvalidator_dispatch(t *testing.T) {
	rec := &recordingCache{}
	inv := NewInvalidator(rec)
	ctx := context.Background()

	alice := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	inv.RoleGranted(ctx, alice)
	if !rec.has(PrefixForPrincipal(NSDomains, alice)) {
		t.Errorf("RoleGranted should drop domains for grantee, prefixes=%v", rec.prefixes)
	}
	if !rec.has(PrefixForPrincipal(NSEffectiveAccess, alice)) {
		t.Errorf("RoleGranted should drop effective-access for grantee, prefixes=%v", rec.prefixes)
	}

	rec.reset()
	inv.PolicyUpdated(ctx)
	if !rec.has(NSEffectiveAccess) || !rec.has(NSDomains) {
		t.Errorf("PolicyUpdated should drop both namespaces, prefixes=%v", rec.prefixes)
	}

	rec.reset()
	inv.EntityPublished(ctx, uuid.New())
	if !rec.has(NSSearchKeyword) {
		t.Errorf("EntityPublished should drop search-keyword, prefixes=%v", rec.prefixes)
	}
}

type recordingCache struct {
	prefixes []string
}

func (r *recordingCache) Get(context.Context, string) ([]byte, error) { return nil, ErrMiss }
func (r *recordingCache) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}
func (r *recordingCache) Delete(context.Context, string) error { return nil }
func (r *recordingCache) DeletePrefix(_ context.Context, p string) error {
	r.prefixes = append(r.prefixes, p)
	return nil
}
func (r *recordingCache) reset() { r.prefixes = nil }
func (r *recordingCache) has(p string) bool {
	for _, s := range r.prefixes {
		if s == p {
			return true
		}
	}
	return false
}
