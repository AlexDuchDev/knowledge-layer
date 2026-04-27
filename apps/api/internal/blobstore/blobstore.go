package blobstore

import (
	"context"
	"io"
)

// BlobStore persists large binary objects (e.g. raw ingestion artifacts) to S3-compatible storage.
// Implementations must not perform authorization; callers enforce feed policy before write.
type BlobStore interface {
	Put(ctx context.Context, objectKey string, contentType string, r io.Reader, size int64) (storageURI string, err error)
}

// Nop returns a URI without persisting (local dev / tests).
type Nop struct{}

func (Nop) Put(_ context.Context, objectKey string, _ string, r io.Reader, _ int64) (string, error) {
	_, _ = io.Copy(io.Discard, r)
	return "nop://" + objectKey, nil
}
