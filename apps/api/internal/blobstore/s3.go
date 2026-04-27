package blobstore

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3BlobStore uploads via the MinIO client (S3-compatible: MinIO, R2, AWS).
type S3BlobStore struct {
	client *minio.Client
	bucket string
}

// NewS3BlobStore dials an S3-compatible endpoint. endpoint is host:port without a scheme when useSSL is false.
func NewS3BlobStore(endpoint, region, bucket, accessKey, secretKey string, useSSL, pathStyle bool) (*S3BlobStore, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" || strings.TrimSpace(bucket) == "" {
		return nil, fmt.Errorf("blobstore: s3 endpoint and bucket required")
	}
	if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" {
		return nil, fmt.Errorf("blobstore: s3 access key and secret key required")
	}
	lookup := minio.BucketLookupDNS
	if pathStyle {
		lookup = minio.BucketLookupPath
	}
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       useSSL,
		Region:       strings.TrimSpace(region),
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("blobstore: minio client: %w", err)
	}
	return &S3BlobStore{client: cli, bucket: bucket}, nil
}

// Put streams the object into the configured bucket.
func (s *S3BlobStore) Put(ctx context.Context, objectKey string, contentType string, r io.Reader, size int64) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("blobstore: nil s3 client")
	}
	key := strings.TrimLeft(strings.TrimSpace(objectKey), "/")
	if key == "" {
		return "", fmt.Errorf("blobstore: empty object key")
	}
	opts := minio.PutObjectOptions{ContentType: contentType}
	var err error
	if size > 0 {
		_, err = s.client.PutObject(ctx, s.bucket, key, r, size, opts)
	} else {
		_, err = s.client.PutObject(ctx, s.bucket, key, r, -1, opts)
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("s3://%s/%s", s.bucket, key), nil
}
