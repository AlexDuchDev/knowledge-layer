package knowledge_core

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Entity struct {
	ID               uuid.UUID  `json:"id"`
	Type             string     `json:"type"`
	Title            string     `json:"title"`
	Summary          *string    `json:"summary,omitempty"`
	Body             *string    `json:"body,omitempty"`
	OwnerID          *uuid.UUID `json:"owner_id,omitempty"`
	DomainID         uuid.UUID  `json:"domain_id"`
	SensitivityLevel int        `json:"sensitivity_level"`
	TruthMode        string     `json:"truth_mode"`
	LifecycleState   string     `json:"lifecycle_state"`
	FreshnessStatus  string     `json:"freshness_status"`
	CanonicalStatus  string     `json:"canonical_status"`
	ApprovalStatus   string     `json:"approval_status"`
	ExternalRef      *string    `json:"external_ref,omitempty"`
	AccessPolicyID   *uuid.UUID `json:"access_policy_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type EntityPayload struct {
	EntityID      uuid.UUID       `json:"entity_id"`
	EntityType    string          `json:"entity_type"`
	PayloadJSON   json.RawMessage `json:"payload_json"`
	SchemaVersion int             `json:"schema_version"`
}

type EntityPayloadWithTimestamps struct {
	EntityID      uuid.UUID       `json:"entity_id"`
	EntityType    string          `json:"entity_type"`
	PayloadJSON   json.RawMessage `json:"payload_json"`
	SchemaVersion int             `json:"schema_version"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type EntityLink struct {
	ID           uuid.UUID `json:"id"`
	FromEntityID uuid.UUID `json:"from_entity_id"`
	RelationType string    `json:"relation_type"`
	ToEntityID   uuid.UUID `json:"to_entity_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type ProvenanceRecord struct {
	ID           uuid.UUID  `json:"id"`
	TargetType   string     `json:"target_type"`
	TargetID     uuid.UUID  `json:"target_id"`
	OriginType   string     `json:"origin_type"`
	OriginRef    *string    `json:"origin_ref,omitempty"`
	SourceFeedID *uuid.UUID `json:"source_feed_id,omitempty"`
	JobRunID     *uuid.UUID `json:"job_run_id,omitempty"`
	CreatedByID  *uuid.UUID `json:"created_by_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ProvenanceEvidence struct {
	Record              ProvenanceRecord `json:"record"`
	RawArtifactIDs      []uuid.UUID      `json:"raw_artifact_ids"`
	NormalizedRecordIDs []uuid.UUID      `json:"normalized_record_ids"`
}

// OnEntityPersistedHook runs after a successful entity create or content patch (best-effort; errors are ignored by callers).
type OnEntityPersistedHook func(ctx context.Context, entityID uuid.UUID) error

type EntityRepo struct {
	pool        *pgxpool.Pool
	onPersisted OnEntityPersistedHook
}

func NewEntityRepo(pool *pgxpool.Pool) *EntityRepo { return &EntityRepo{pool: pool} }

// SetOnEntityPersisted registers a callback for chunk rebuild / embedding enqueue after entity writes.
func (r *EntityRepo) SetOnEntityPersisted(h OnEntityPersistedHook) {
	r.onPersisted = h
}

func (r *EntityRepo) fireEntityPersisted(ctx context.Context, entityID uuid.UUID) {
	if r == nil || r.onPersisted == nil {
		return
	}
	_ = r.onPersisted(ctx, entityID)
}

func (r *EntityRepo) List(ctx context.Context, filters map[string]string) ([]Entity, error) {
	q := `SELECT id, type, title, summary, body, owner_id, domain_id, sensitivity_level, truth_mode,
		lifecycle_state, freshness_status, canonical_status, approval_status, external_ref, access_policy_id, created_at, updated_at
		FROM entities WHERE archived_at IS NULL`
	args := []any{}
	n := 1
	if v := filters["type"]; v != "" {
		q += ` AND type = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["domain_id"]; v != "" {
		id, _ := uuid.Parse(v)
		q += ` AND domain_id = $` + strconv.Itoa(n)
		args = append(args, id)
		n++
	}
	if v := filters["truth_mode"]; v != "" {
		q += ` AND truth_mode = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["lifecycle_state"]; v != "" {
		q += ` AND lifecycle_state = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["approval_status"]; v != "" {
		q += ` AND approval_status = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["owner_id"]; v != "" {
		id, _ := uuid.Parse(v)
		q += ` AND owner_id = $` + strconv.Itoa(n)
		args = append(args, id)
		n++
	}
	q += ` ORDER BY updated_at DESC LIMIT 500`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntities(rows)
}

// ListInDomains lists non-archived entities restricted to the given domain IDs (typically caller's granted domains).
func (r *EntityRepo) ListInDomains(ctx context.Context, filters map[string]string, domainIDs []uuid.UUID, limit, offset int) ([]Entity, error) {
	if len(domainIDs) == 0 {
		return []Entity{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	q := `SELECT id, type, title, summary, body, owner_id, domain_id, sensitivity_level, truth_mode,
		lifecycle_state, freshness_status, canonical_status, approval_status, external_ref, access_policy_id, created_at, updated_at
		FROM entities WHERE archived_at IS NULL AND domain_id = ANY($1)`
	args := []any{domainIDs}
	n := 2
	if v := filters["type"]; v != "" {
		q += ` AND type = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["domain_id"]; v != "" {
		id, _ := uuid.Parse(v)
		q += ` AND domain_id = $` + strconv.Itoa(n)
		args = append(args, id)
		n++
	}
	if v := filters["truth_mode"]; v != "" {
		q += ` AND truth_mode = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["lifecycle_state"]; v != "" {
		q += ` AND lifecycle_state = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["approval_status"]; v != "" {
		q += ` AND approval_status = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["owner_id"]; v != "" {
		id, _ := uuid.Parse(v)
		q += ` AND owner_id = $` + strconv.Itoa(n)
		args = append(args, id)
		n++
	}
	orderCol := "updated_at"
	if filters["sort"] == "created_at" {
		orderCol = "created_at"
	}
	q += ` ORDER BY ` + orderCol + ` DESC LIMIT $` + strconv.Itoa(n) + ` OFFSET $` + strconv.Itoa(n+1)
	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntities(rows)
}

func scanEntities(rows pgx.Rows) ([]Entity, error) {
	var list []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.Type, &e.Title, &e.Summary, &e.Body, &e.OwnerID, &e.DomainID,
			&e.SensitivityLevel, &e.TruthMode, &e.LifecycleState, &e.FreshnessStatus,
			&e.CanonicalStatus, &e.ApprovalStatus, &e.ExternalRef, &e.AccessPolicyID, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func (r *EntityRepo) Get(ctx context.Context, id uuid.UUID) (*Entity, error) {
	var e Entity
	err := r.pool.QueryRow(ctx, `
		SELECT id, type, title, summary, body, owner_id, domain_id, sensitivity_level, truth_mode,
			lifecycle_state, freshness_status, canonical_status, approval_status, external_ref, access_policy_id, created_at, updated_at
		FROM entities WHERE id = $1 AND archived_at IS NULL`, id,
	).Scan(&e.ID, &e.Type, &e.Title, &e.Summary, &e.Body, &e.OwnerID, &e.DomainID,
		&e.SensitivityLevel, &e.TruthMode, &e.LifecycleState, &e.FreshnessStatus,
		&e.CanonicalStatus, &e.ApprovalStatus, &e.ExternalRef, &e.AccessPolicyID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EntityRepo) GetPayload(ctx context.Context, entityID uuid.UUID) (*EntityPayloadWithTimestamps, error) {
	var p EntityPayloadWithTimestamps
	err := r.pool.QueryRow(ctx, `
		SELECT entity_id, entity_type, payload_json, schema_version, created_at, updated_at
		FROM entity_payloads WHERE entity_id = $1`, entityID,
	).Scan(&p.EntityID, &p.EntityType, &p.PayloadJSON, &p.SchemaVersion, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type CreateEntityInput struct {
	Type             string          `json:"type"`
	Title            string          `json:"title"`
	Summary          *string         `json:"summary,omitempty"`
	Body             *string         `json:"body,omitempty"`
	OwnerID          *uuid.UUID      `json:"owner_id,omitempty"`
	DomainID         uuid.UUID       `json:"domain_id"`
	SensitivityLevel int             `json:"sensitivity_level"`
	TruthMode        string          `json:"truth_mode"`
	LifecycleState   string          `json:"lifecycle_state"`
	ExternalRef      *string         `json:"external_ref,omitempty"`
	PayloadJSON      json.RawMessage `json:"payload_json,omitempty"`
}

func (r *EntityRepo) Create(ctx context.Context, in CreateEntityInput) (*Entity, error) {
	if in.TruthMode == "" {
		in.TruthMode = "derived"
	}
	if in.LifecycleState == "" {
		in.LifecycleState = "draft"
	}
	id := uuid.New()
	now := time.Now().UTC()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var e Entity
	err = tx.QueryRow(ctx, `
		INSERT INTO entities (id, type, title, summary, body, owner_id, domain_id, sensitivity_level, truth_mode,
			lifecycle_state, freshness_status, canonical_status, approval_status, external_ref, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'unknown','draft','none',$11,$12,$12)
		RETURNING id, type, title, summary, body, owner_id, domain_id, sensitivity_level, truth_mode,
			lifecycle_state, freshness_status, canonical_status, approval_status, external_ref, access_policy_id, created_at, updated_at`,
		id, in.Type, in.Title, in.Summary, in.Body, in.OwnerID, in.DomainID, in.SensitivityLevel,
		in.TruthMode, in.LifecycleState, in.ExternalRef, now, now,
	).Scan(&e.ID, &e.Type, &e.Title, &e.Summary, &e.Body, &e.OwnerID, &e.DomainID,
		&e.SensitivityLevel, &e.TruthMode, &e.LifecycleState, &e.FreshnessStatus,
		&e.CanonicalStatus, &e.ApprovalStatus, &e.ExternalRef, &e.AccessPolicyID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if len(in.PayloadJSON) > 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO entity_payloads (entity_id, entity_type, payload_json, schema_version, created_at, updated_at)
			VALUES ($1,$2,$3,1,$4,$4)`, e.ID, in.Type, in.PayloadJSON, now)
		if err != nil {
			return nil, err
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO entity_versions (entity_id, entity_type, version_number, snapshot_json, change_summary, created_at)
		VALUES ($1,$2,1,$3,'create',$4)`, e.ID, in.Type, snapshotEntity(e), now)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO entity_search_projection (entity_id, domain_id, owner_id, entity_type, truth_mode, lifecycle_state, freshness_status, title, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		e.ID, e.DomainID, e.OwnerID, e.Type, e.TruthMode, e.LifecycleState, e.FreshnessStatus, e.Title, e.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	r.fireEntityPersisted(ctx, e.ID)
	return &e, nil
}

func snapshotEntity(e Entity) []byte {
	b, _ := json.Marshal(e)
	return b
}

type PatchEntityInput struct {
	Title          *string `json:"title,omitempty"`
	Summary        *string `json:"summary,omitempty"`
	Body           *string `json:"body,omitempty"`
	TruthMode      *string `json:"truth_mode,omitempty"`
	LifecycleState *string `json:"lifecycle_state,omitempty"`
}

// ErrPatchPublishForbidden is returned when a Patch tries to set
// lifecycle_state="published". Use EntityRepo.Publish instead so the version
// snapshot, approval stamps, audit emission, and search projection update
// happen via the single canonical path (Phase 4.2.1).
var ErrPatchPublishForbidden = errors.New("entity: PATCH cannot set lifecycle_state=published; use the publish flow")

func (r *EntityRepo) Patch(ctx context.Context, id uuid.UUID, in PatchEntityInput) (*Entity, error) {
	if in.LifecycleState != nil && *in.LifecycleState == "published" {
		return nil, ErrPatchPublishForbidden
	}
	e, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Title != nil {
		e.Title = *in.Title
	}
	if in.Summary != nil {
		e.Summary = in.Summary
	}
	if in.Body != nil {
		e.Body = in.Body
	}
	if in.TruthMode != nil {
		e.TruthMode = *in.TruthMode
	}
	if in.LifecycleState != nil {
		e.LifecycleState = *in.LifecycleState
	}
	now := time.Now().UTC()
	e.UpdatedAt = now

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE entities SET title=$2, summary=$3, body=$4, truth_mode=$5, lifecycle_state=$6, updated_at=$7
		WHERE id=$1`, id, e.Title, e.Summary, e.Body, e.TruthMode, e.LifecycleState, now)
	if err != nil {
		return nil, err
	}

	var vn int
	_ = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version_number),0)+1 FROM entity_versions WHERE entity_id=$1`, id).Scan(&vn)
	_, err = tx.Exec(ctx, `
		INSERT INTO entity_versions (entity_id, entity_type, version_number, snapshot_json, change_summary, created_at)
		VALUES ($1,$2,$3,$4,'patch',$5)`, id, e.Type, vn, snapshotEntity(*e), now)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE entity_search_projection SET truth_mode=$2, lifecycle_state=$3, title=$4, updated_at=$5
		WHERE entity_id=$1`, id, e.TruthMode, e.LifecycleState, e.Title, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	r.fireEntityPersisted(ctx, id)
	return r.Get(ctx, id)
}

func (r *EntityRepo) AddLink(ctx context.Context, fromID uuid.UUID, toID uuid.UUID, relation string, actor *uuid.UUID) (*EntityLink, error) {
	var fc, tc string
	if err := r.pool.QueryRow(ctx, `SELECT type FROM entities WHERE id=$1`, fromID).Scan(&fc); err != nil {
		return nil, err
	}
	if err := r.pool.QueryRow(ctx, `SELECT type FROM entities WHERE id=$1`, toID).Scan(&tc); err != nil {
		return nil, err
	}
	id := uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO entity_links (id, from_entity_id, from_entity_type, relation_type, to_entity_id, to_entity_type, created_by_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,now())`,
		id, fromID, fc, relation, toID, tc, actor)
	if err != nil {
		return nil, err
	}
	return &EntityLink{ID: id, FromEntityID: fromID, ToEntityID: toID, RelationType: relation, CreatedAt: time.Now().UTC()}, nil
}

func (r *EntityRepo) ListLinks(ctx context.Context, entityID uuid.UUID) ([]EntityLink, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, from_entity_id, relation_type, to_entity_id, created_at FROM entity_links
		WHERE from_entity_id = $1 OR to_entity_id = $1`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []EntityLink
	for rows.Next() {
		var l EntityLink
		if err := rows.Scan(&l.ID, &l.FromEntityID, &l.RelationType, &l.ToEntityID, &l.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

func (r *EntityRepo) AttachProvenance(ctx context.Context, record ProvenanceRecord, rawIDs []uuid.UUID, normIDs []uuid.UUID) (*ProvenanceRecord, error) {
	id := uuid.New()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		INSERT INTO provenance_records (id, target_type, target_id, origin_type, origin_ref, source_feed_id, job_run_id, created_by_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8, now())`,
		id, record.TargetType, record.TargetID, record.OriginType, record.OriginRef, record.SourceFeedID, record.JobRunID, record.CreatedByID)
	if err != nil {
		return nil, err
	}
	for _, rid := range rawIDs {
		_, err = tx.Exec(ctx, `INSERT INTO provenance_raw_artifacts (provenance_record_id, raw_artifact_id) VALUES ($1,$2)`, id, rid)
		if err != nil {
			return nil, err
		}
	}
	for _, nid := range normIDs {
		_, err = tx.Exec(ctx, `INSERT INTO provenance_normalized_records (provenance_record_id, normalized_record_id) VALUES ($1,$2)`, id, nid)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	record.ID = id
	record.CreatedAt = time.Now().UTC()
	return &record, nil
}

func (r *EntityRepo) ListProvenance(ctx context.Context, entityID uuid.UUID) ([]ProvenanceRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, target_type, target_id, origin_type, origin_ref, source_feed_id, job_run_id, created_by_id, created_at
		FROM provenance_records WHERE target_type='entity' AND target_id=$1 ORDER BY created_at ASC`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ProvenanceRecord
	for rows.Next() {
		var p ProvenanceRecord
		if err := rows.Scan(&p.ID, &p.TargetType, &p.TargetID, &p.OriginType, &p.OriginRef, &p.SourceFeedID, &p.JobRunID, &p.CreatedByID, &p.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *EntityRepo) ListProvenanceEvidence(ctx context.Context, entityID uuid.UUID) ([]ProvenanceEvidence, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pr.id, pr.target_type, pr.target_id, pr.origin_type, pr.origin_ref, pr.source_feed_id, pr.job_run_id, pr.created_by_id, pr.created_at,
			   pra.raw_artifact_id,
			   pnr.normalized_record_id
		FROM provenance_records pr
		LEFT JOIN provenance_raw_artifacts pra ON pra.provenance_record_id = pr.id
		LEFT JOIN provenance_normalized_records pnr ON pnr.provenance_record_id = pr.id
		WHERE pr.target_type='entity' AND pr.target_id=$1
		ORDER BY pr.created_at ASC`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type agg struct {
		rec  ProvenanceRecord
		raw  map[uuid.UUID]struct{}
		norm map[uuid.UUID]struct{}
	}
	m := map[uuid.UUID]*agg{}
	order := []uuid.UUID{}

	for rows.Next() {
		var pr ProvenanceRecord
		var rawID *uuid.UUID
		var normID *uuid.UUID
		if err := rows.Scan(&pr.ID, &pr.TargetType, &pr.TargetID, &pr.OriginType, &pr.OriginRef, &pr.SourceFeedID, &pr.JobRunID, &pr.CreatedByID, &pr.CreatedAt, &rawID, &normID); err != nil {
			return nil, err
		}
		a, ok := m[pr.ID]
		if !ok {
			a = &agg{rec: pr, raw: map[uuid.UUID]struct{}{}, norm: map[uuid.UUID]struct{}{}}
			m[pr.ID] = a
			order = append(order, pr.ID)
		}
		if rawID != nil {
			a.raw[*rawID] = struct{}{}
		}
		if normID != nil {
			a.norm[*normID] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]ProvenanceEvidence, 0, len(order))
	for _, id := range order {
		a := m[id]
		raw := make([]uuid.UUID, 0, len(a.raw))
		for rid := range a.raw {
			raw = append(raw, rid)
		}
		norm := make([]uuid.UUID, 0, len(a.norm))
		for nid := range a.norm {
			norm = append(norm, nid)
		}
		out = append(out, ProvenanceEvidence{
			Record:              a.rec,
			RawArtifactIDs:      raw,
			NormalizedRecordIDs: norm,
		})
	}
	return out, nil
}

// EntityVersion is a version snapshot row for an entity.
type EntityVersion struct {
	ID            uuid.UUID       `json:"id"`
	EntityID      uuid.UUID       `json:"entity_id"`
	EntityType    string          `json:"entity_type"`
	VersionNumber int             `json:"version_number"`
	SnapshotJSON  json.RawMessage `json:"snapshot_json"`
	ChangeSummary *string         `json:"change_summary,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// EntityCompiledTruth stores the “compiled truth” projection for an entity (top-of-page summary/body),
// while the append-only timeline is represented by entity_versions.
type EntityCompiledTruth struct {
	EntityID             uuid.UUID  `json:"entity_id"`
	CompiledSummary      *string    `json:"compiled_summary,omitempty"`
	CompiledBody         *string    `json:"compiled_body,omitempty"`
	BasedOnVersionNumber *int       `json:"based_on_version_number,omitempty"`
	CompiledByType       *string    `json:"compiled_by_type,omitempty"`
	CompiledByID         *uuid.UUID `json:"compiled_by_id,omitempty"`
	CompiledAt           time.Time  `json:"compiled_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (r *EntityRepo) GetCompiledTruth(ctx context.Context, entityID uuid.UUID) (*EntityCompiledTruth, error) {
	var ct EntityCompiledTruth
	err := r.pool.QueryRow(ctx, `
		SELECT entity_id, compiled_summary, compiled_body, based_on_version_number, compiled_by_type, compiled_by_id, compiled_at, updated_at
		FROM entity_compiled_truth
		WHERE entity_id=$1`, entityID,
	).Scan(&ct.EntityID, &ct.CompiledSummary, &ct.CompiledBody, &ct.BasedOnVersionNumber, &ct.CompiledByType, &ct.CompiledByID, &ct.CompiledAt, &ct.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &ct, nil
}

func (r *EntityRepo) UpsertCompiledTruth(ctx context.Context, in EntityCompiledTruth) error {
	if in.EntityID == uuid.Nil {
		return errors.New("compiled truth: entity_id required")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO entity_compiled_truth (entity_id, compiled_summary, compiled_body, based_on_version_number, compiled_by_type, compiled_by_id, compiled_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,now(),now())
		ON CONFLICT (entity_id) DO UPDATE SET
			compiled_summary=EXCLUDED.compiled_summary,
			compiled_body=EXCLUDED.compiled_body,
			based_on_version_number=EXCLUDED.based_on_version_number,
			compiled_by_type=EXCLUDED.compiled_by_type,
			compiled_by_id=EXCLUDED.compiled_by_id,
			compiled_at=now(),
			updated_at=now()`,
		in.EntityID,
		in.CompiledSummary,
		in.CompiledBody,
		in.BasedOnVersionNumber,
		in.CompiledByType,
		in.CompiledByID,
	)
	return err
}

func (r *EntityRepo) ListEntityVersions(ctx context.Context, entityID uuid.UUID) ([]EntityVersion, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, entity_id, entity_type, version_number, snapshot_json, change_summary, created_at
		FROM entity_versions WHERE entity_id = $1 ORDER BY version_number ASC`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []EntityVersion
	for rows.Next() {
		var v EntityVersion
		if err := rows.Scan(&v.ID, &v.EntityID, &v.EntityType, &v.VersionNumber, &v.SnapshotJSON, &v.ChangeSummary, &v.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func (r *EntityRepo) GetEntityVersion(ctx context.Context, versionID uuid.UUID) (*EntityVersion, error) {
	var v EntityVersion
	err := r.pool.QueryRow(ctx, `
		SELECT id, entity_id, entity_type, version_number, snapshot_json, change_summary, created_at
		FROM entity_versions WHERE id = $1`, versionID,
	).Scan(&v.ID, &v.EntityID, &v.EntityType, &v.VersionNumber, &v.SnapshotJSON, &v.ChangeSummary, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// PromoteDerivedToCanonicalPlatform marks a reviewed derived entity as canonical_in_platform (explicit promotion only).
func (r *EntityRepo) PromoteDerivedToCanonicalPlatform(ctx context.Context, entityID uuid.UUID, actor uuid.UUID, summary string) (*Entity, error) {
	e, err := r.Get(ctx, entityID)
	if err != nil {
		return nil, err
	}
	if e.TruthMode != "derived" {
		return nil, errors.New("promotion only allowed from truth_mode=derived")
	}
	now := time.Now().UTC()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	e.TruthMode = "canonical_in_platform"
	e.CanonicalStatus = "approved"
	e.ApprovalStatus = "approved"
	e.UpdatedAt = now

	_, err = tx.Exec(ctx, `
		UPDATE entities SET truth_mode=$2, canonical_status=$3, approval_status=$4,
			approved_at=$5, approved_by_id=$6, updated_at=$7
		WHERE id=$1 AND truth_mode='derived' AND archived_at IS NULL`,
		entityID, e.TruthMode, e.CanonicalStatus, e.ApprovalStatus, now, actor, now)
	if err != nil {
		return nil, err
	}

	var vn int
	_ = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version_number),0)+1 FROM entity_versions WHERE entity_id=$1`, entityID).Scan(&vn)
	change := "promote_to_canonical_platform"
	if summary != "" {
		change = summary
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO entity_versions (entity_id, entity_type, version_number, snapshot_json, change_summary, changed_by_type, changed_by_id, created_at)
		VALUES ($1,$2,$3,$4,$5,'user',$6,$7)`,
		entityID, e.Type, vn, snapshotEntity(*e), change, actor, now)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE entity_search_projection SET truth_mode=$2, updated_at=$3 WHERE entity_id=$1`,
		entityID, e.TruthMode, now)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.Get(ctx, entityID)
}

var ErrNotFound = errors.New("not found")

// ErrAlreadyPublished is returned by Publish when the entity is already in
// `lifecycle_state='published'`. Re-publish is a no-op rather than an error
// only because some callers (review approval, automated workflows) may
// idempotently call Publish; this sentinel lets HTTP routes distinguish
// "did nothing" from "promoted to published" when shaping the response.
var ErrAlreadyPublished = errors.New("entity already published")

// PublishResult is the outcome of EntityRepo.Publish.
type PublishResult struct {
	Entity        *Entity
	WasPublished  bool // true when this call moved lifecycle_state to "published"
	WasIdempotent bool // true when the entity was already published; Entity is the unchanged row
}

// Publish is the SINGLE canonical path for transitioning an entity to
// lifecycle_state="published" (Phase 4.2.1).
//
// It atomically:
//   - Moves lifecycle_state to "published" if not already there.
//   - Stamps approval_status="approved", approved_at=now, approved_by_id=principal.
//   - Inserts an entity_versions snapshot ("publish").
//   - Updates entity_search_projection so search reflects the new lifecycle.
//
// Routes that need to publish MUST go through this method. The PATCH
// /entities/:id route explicitly rejects in.LifecycleState="published" and
// directs callers here so the version-snapshot + projection updates are
// guaranteed and so audit emission has a single chokepoint.
//
// What this does NOT do (caller's responsibility):
//   - Permission check — caller decides whether the principal may publish in
//     the entity's domain (`AccessEvaluator.Evaluate(action="publish")`).
//   - Approval-queue gate — caller (typically POST /review-tasks/:id/approve)
//     decides whether a review task must precede direct-publish.
//   - Search reindex (`d.Search.ReindexEntity`) — caller hooks the OpenSearch
//     reindex after the SQL transaction commits.
//   - Audit emission (`audit.Write event_type="entity.published"`) — caller
//     emits with the right principal + correlation context.
//
// This split keeps the repo layer focused on durable state changes while
// route handlers retain the security and observability concerns they're
// already wired for.
func (r *EntityRepo) Publish(ctx context.Context, entityID, principal uuid.UUID) (*PublishResult, error) {
	if entityID == uuid.Nil {
		return nil, errors.New("publish: entity_id required")
	}
	if principal == uuid.Nil {
		return nil, errors.New("publish: principal required")
	}
	e, err := r.Get(ctx, entityID)
	if err != nil {
		return nil, err
	}
	if e.LifecycleState == "published" {
		return &PublishResult{Entity: e, WasPublished: false, WasIdempotent: true}, nil
	}

	now := time.Now().UTC()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE entities
		SET lifecycle_state='published',
		    approval_status='approved',
		    approved_at=$2,
		    approved_by_id=$3,
		    updated_at=$2
		WHERE id=$1 AND archived_at IS NULL`,
		entityID, now, principal)
	if err != nil {
		return nil, err
	}

	e.LifecycleState = "published"
	e.ApprovalStatus = "approved"
	e.UpdatedAt = now

	var vn int
	_ = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version_number),0)+1 FROM entity_versions WHERE entity_id=$1`, entityID).Scan(&vn)
	_, err = tx.Exec(ctx, `
		INSERT INTO entity_versions (entity_id, entity_type, version_number, snapshot_json, change_summary, changed_by_type, changed_by_id, created_at)
		VALUES ($1,$2,$3,$4,'publish','user',$5,$6)`,
		entityID, e.Type, vn, snapshotEntity(*e), principal, now)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE entity_search_projection
		SET lifecycle_state='published', updated_at=$2
		WHERE entity_id=$1`, entityID, now)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &PublishResult{Entity: e, WasPublished: true, WasIdempotent: false}, nil
}
