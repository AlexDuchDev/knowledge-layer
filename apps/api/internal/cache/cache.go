// Package cache provides a small abstraction for in-process L1 caching
// (BigCache) and optional out-of-process L2 (Redis), used for hot read paths
// like /domains, /users/:id/effective-access, /search keyword, and
// /knowledge-jobs/engine-metadata.
//
// Inspired by Hugr's gocache integration. The intentional surface is very
// small — Get / Set / Delete / DeletePrefix — so callers don't accidentally
// build complex cache logic in handlers. All keys go through keys.go for
// principal-scoping safety.
package cache

import (
	"context"
	"errors"
	"time"
)

// ErrMiss is returned by Get when the key is not present. Callers should treat
// this as a soft signal (compute + Set) rather than an error to log.
var ErrMiss = errors.New("cache: miss")

// Cache is the interface every cache backend implements. Disabled caches
// return ErrMiss on Get and silently no-op on Set / Delete / DeletePrefix.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	// DeletePrefix removes every key sharing the given prefix. Used by the
	// invalidator to drop entire namespaces (e.g. all "search:..." keys
	// when a feed config changes).
	DeletePrefix(ctx context.Context, prefix string) error
}

// nullCache is the zero-value cache: every Get is a miss, every Set is a
// no-op. Composed when CACHE_L1_ENABLED=false so handlers can call cache
// methods unconditionally.
type nullCache struct{}

// Null returns a cache that never stores anything. Safe to inject when the
// operator has caching disabled or during tests.
func Null() Cache { return nullCache{} }

func (nullCache) Get(context.Context, string) ([]byte, error) { return nil, ErrMiss }

func (nullCache) Set(context.Context, string, []byte, time.Duration) error { return nil }

func (nullCache) Delete(context.Context, string) error { return nil }

func (nullCache) DeletePrefix(context.Context, string) error { return nil }
