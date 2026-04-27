package blobstore

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/knowledgelayer/api/internal/config"
)

func TestFromConfig_NopDefault(t *testing.T) {
	bs, err := FromConfig(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := bs.(Nop); !ok {
		t.Fatalf("expected Nop, got %T", bs)
	}
	uri, err := bs.Put(context.Background(), "k/obj", "text/plain", bytes.NewReader([]byte("hi")), 2)
	if err != nil {
		t.Fatal(err)
	}
	if uri != "nop://k/obj" {
		t.Fatalf("uri: %q", uri)
	}
}

func TestFromConfig_S3RequiresFields(t *testing.T) {
	_, err := FromConfig(config.Config{
		BlobStoreBackend: "s3",
	})
	if err == nil {
		t.Fatal("expected error for incomplete s3 config")
	}
	if !strings.Contains(err.Error(), "blobstore") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestFromConfig_UnknownBackend(t *testing.T) {
	_, err := FromConfig(config.Config{BlobStoreBackend: "cosmic"})
	if err == nil {
		t.Fatal("expected error")
	}
}
