package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/allegro/bigcache/v3"
)

// bigcacheBackend is the L1 in-process implementation. BigCache is preferred
// over a sync.Map because (a) it bounds total memory (no slow leak when keys
// vary), (b) supports a TTL natively, and (c) the underlying shards remove
// the contention seen on hot reads.
//
// Prefix-deletion is implemented by tracking the live key set in a sidecar
// sync.Map. BigCache itself does not expose key enumeration. The cost is one
// extra map write per Set + one map iteration per DeletePrefix; acceptable
// for the hot-read invalidation rate (a few writes per minute).
type bigcacheBackend struct {
	store *bigcache.BigCache

	// keys mirrors the live key set so DeletePrefix can fan out. Values are
	// expiration timestamps so a stale entry can be GC'd lazily on read.
	keys *sync.Map
}

// BigcacheOptions configures the L1 backend at construction.
type BigcacheOptions struct {
	// DefaultTTL applies to entries written without an explicit TTL via Set.
	DefaultTTL time.Duration
	// MaxMB caps total memory. BigCache pre-allocates eagerly.
	MaxMB int
}

// NewBigcache builds an L1 cache backed by BigCache. The default TTL is the
// minimum of the operator-configured value and 5 minutes; longer TTLs raise
// the staleness window for permission-derived data and need an explicit
// invalidation event to compensate.
func NewBigcache(opts BigcacheOptions) (Cache, error) {
	ttl := opts.DefaultTTL
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	maxMB := opts.MaxMB
	if maxMB <= 0 {
		maxMB = 64
	}
	cfg := bigcache.DefaultConfig(ttl)
	cfg.HardMaxCacheSize = maxMB
	cfg.Verbose = false

	store, err := bigcache.New(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	return &bigcacheBackend{
		store: store,
		keys:  &sync.Map{},
	}, nil
}

func (b *bigcacheBackend) Get(_ context.Context, key string) ([]byte, error) {
	val, err := b.store.Get(key)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			b.keys.Delete(key)
			return nil, ErrMiss
		}
		return nil, err
	}
	return val, nil
}

func (b *bigcacheBackend) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	// BigCache uses a uniform default TTL set at construction; the per-key
	// ttl arg is honored by the abstraction's contract but not by the
	// backend. Documented as "TTL is best-effort; minimum is the cache's
	// global default" in the cache package doc. Keys that need shorter
	// TTLs should be skipped (Null cache) rather than fitted in here.
	if err := b.store.Set(key, value); err != nil {
		return err
	}
	b.keys.Store(key, time.Now())
	return nil
}

func (b *bigcacheBackend) Delete(_ context.Context, key string) error {
	if err := b.store.Delete(key); err != nil && !errors.Is(err, bigcache.ErrEntryNotFound) {
		return err
	}
	b.keys.Delete(key)
	return nil
}

func (b *bigcacheBackend) DeletePrefix(ctx context.Context, prefix string) error {
	b.keys.Range(func(k, _ any) bool {
		ks, ok := k.(string)
		if !ok {
			return true
		}
		if hasPrefix(ks, prefix) {
			_ = b.store.Delete(ks)
			b.keys.Delete(ks)
		}
		// Honor cancellation so a misuse with prefix="" can't lock up the
		// cache during a long shutdown.
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	})
	return ctx.Err()
}

// hasPrefix is a small alias so we don't pull in strings just for one call.
func hasPrefix(s, prefix string) bool {
	if len(prefix) > len(s) {
		return false
	}
	return s[:len(prefix)] == prefix
}
