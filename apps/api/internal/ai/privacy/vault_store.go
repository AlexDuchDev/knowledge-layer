package privacy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/knowledgelayer/api/internal/audit"
)

// VaultEntry is one row to persist for a correlation id (job run or ask trace).
type VaultEntry struct {
	Placeholder        string
	EntityType         SensitiveEntityType
	Value              string
	PolicySnapshotJSON json.RawMessage
}

// DecryptedMapping is a vault row decrypted for an internal rehydration step only.
type DecryptedMapping struct {
	Placeholder string
	EntityType  SensitiveEntityType
	Value       string
}

// SecureEntityMapStore persists placeholder mappings encrypted at rest.
//
// All vault operations (PutBatch, ListDecryptedForCorrelation) emit audit
// events via the embedded audit.Service so that every encrypt/decrypt pass
// leaves a tamper-evident trail. Audit is a hard requirement for production:
// passing a nil audit service is allowed only in tests and degrades to a no-op
// log line (compliance-sensitive deployments must wire it).
type SecureEntityMapStore struct {
	pool  *pgxpool.Pool
	codec *VaultCodec
	audit *audit.Service
}

// NewSecureEntityMapStore constructs the store. Pass auditSvc=nil only in tests.
func NewSecureEntityMapStore(pool *pgxpool.Pool, codec *VaultCodec, auditSvc *audit.Service) *SecureEntityMapStore {
	return &SecureEntityMapStore{pool: pool, codec: codec, audit: auditSvc}
}

// PutBatch writes encrypted mappings with a shared TTL and emits a
// `vault.placeholder_stored` audit event with placeholder count and TTL.
func (s *SecureEntityMapStore) PutBatch(ctx context.Context, correlationID string, principalID *uuid.UUID, jobRunID *uuid.UUID, ttl time.Duration, entries []VaultEntry) error {
	if s == nil || s.pool == nil || s.codec == nil {
		return fmt.Errorf("privacy vault: nil store")
	}
	if correlationID == "" {
		return fmt.Errorf("privacy vault: correlation_id required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	exp := time.Now().UTC().Add(ttl)
	for _, e := range entries {
		nonce, ct, err := s.codec.Encrypt(e.Value)
		if err != nil {
			return err
		}
		snap := e.PolicySnapshotJSON
		if len(snap) == 0 {
			snap = []byte("{}")
		}
		_, err = s.pool.Exec(ctx, `
			INSERT INTO ai_placeholder_mappings (
				correlation_id, principal_id, job_run_id, placeholder, entity_type, nonce, ciphertext, policy_snapshot_json, expires_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			correlationID, principalID, jobRunID, e.Placeholder, string(e.EntityType), nonce, ct, snap, exp,
		)
		if err != nil {
			return err
		}
	}
	s.writeAudit(ctx, "vault.placeholder_stored", principalID, map[string]any{
		"correlation_id":    correlationID,
		"placeholder_count": len(entries),
		"ttl_seconds":       int64(ttl.Seconds()),
		"job_run_id":        formatUUIDPtr(jobRunID),
	})
	return nil
}

// ListDecryptedForCorrelation loads and decrypts all non-expired rows for a correlation id.
// Call only from trusted internal services (rehydration); never expose values to API clients.
// Emits a `vault.placeholder_decrypted` audit event with the row count.
func (s *SecureEntityMapStore) ListDecryptedForCorrelation(ctx context.Context, correlationID string) ([]DecryptedMapping, error) {
	if s == nil || s.pool == nil || s.codec == nil {
		return nil, fmt.Errorf("privacy vault: nil store")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT placeholder, entity_type, nonce, ciphertext
		FROM ai_placeholder_mappings
		WHERE correlation_id = $1 AND expires_at > now()
		ORDER BY created_at ASC`, correlationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []DecryptedMapping
	for rows.Next() {
		var ph, et string
		var nonce, ct []byte
		if err := rows.Scan(&ph, &et, &nonce, &ct); err != nil {
			return nil, err
		}
		val, err := s.codec.Decrypt(nonce, ct)
		if err != nil {
			return nil, err
		}
		list = append(list, DecryptedMapping{
			Placeholder: ph,
			EntityType:  SensitiveEntityType(et),
			Value:       val,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.writeAudit(ctx, "vault.placeholder_decrypted", nil, map[string]any{
		"correlation_id": correlationID,
		"count":          len(list),
	})
	return list, nil
}

// writeAudit best-effort writes a vault audit event. Errors are intentionally
// swallowed so audit failure does not block vault operations; production
// environments should monitor audit_events for `vault.*` event coverage.
func (s *SecureEntityMapStore) writeAudit(ctx context.Context, eventType string, principalID *uuid.UUID, meta map[string]any) {
	if s == nil || s.audit == nil {
		return
	}
	metaJSON, _ := json.Marshal(meta)
	_ = s.audit.Write(ctx, audit.WriteInput{
		EventType:    eventType,
		ActorType:    "system",
		ActorID:      principalID,
		TargetType:   "placeholder_mapping",
		MetadataJSON: metaJSON,
	})
}

func formatUUIDPtr(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}
