package db

import (
	"os"
	"testing"
)

// TestMigrateUp_Idempotent proves embedded migrations apply on a fresh Postgres with pgvector
// (CREATE EXTENSION vector). CI sets DATABASE_URL to the GitHub Actions postgres service.
// Second MigrateUp must succeed (no duplicate version errors).
func TestMigrateUp_Idempotent(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	if err := MigrateUp(dsn); err != nil {
		t.Fatalf("first MigrateUp: %v", err)
	}
	if err := MigrateUp(dsn); err != nil {
		t.Fatalf("second MigrateUp (idempotent): %v", err)
	}
}
