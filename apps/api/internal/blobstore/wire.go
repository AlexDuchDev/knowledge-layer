package blobstore

import (
	"fmt"
	"strings"

	"github.com/knowledgelayer/api/internal/config"
)

// FromConfig returns Nop by default or an S3-compatible store when BLOBSTORE_BACKEND=s3.
func FromConfig(cfg config.Config) (BlobStore, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.BlobStoreBackend)) {
	case "", "nop":
		return Nop{}, nil
	case "s3":
		return NewS3BlobStore(
			cfg.BlobStoreS3Endpoint,
			cfg.BlobStoreS3Region,
			cfg.BlobStoreS3Bucket,
			cfg.BlobStoreS3AccessKey,
			cfg.BlobStoreS3SecretKey,
			cfg.BlobStoreS3UseSSL,
			cfg.BlobStoreS3PathStyle,
		)
	default:
		return nil, fmt.Errorf("blobstore: unknown BLOBSTORE_BACKEND %q", cfg.BlobStoreBackend)
	}
}
