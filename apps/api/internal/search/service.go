package search

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/opensearch"
	"github.com/knowledgelayer/api/internal/platform/permissions"
)

type Hit struct {
	EntityID        uuid.UUID  `json:"entity_id"`
	DomainID        uuid.UUID  `json:"domain_id"`
	DomainName      string     `json:"domain_name,omitempty"`
	OwnerID         *uuid.UUID `json:"owner_id,omitempty"`
	OwnerName       string     `json:"owner_name,omitempty"`
	EntityType      string     `json:"entity_type"`
	Title           string     `json:"title"`
	TruthMode       string     `json:"truth_mode"`
	LifecycleState  string     `json:"lifecycle_state"`
	FreshnessStatus string     `json:"freshness_status"`
	ApprovalStatus  string     `json:"approval_status,omitempty"`
	TrustSummary    string     `json:"trust_summary"`
	// Snippet is set when search used OpenSearch (short excerpt from indexed text).
	Snippet string `json:"snippet,omitempty"`
	// RelationExpansion is set when this row was included via explicit entity_links from another hit (1-hop, same grant domains only).
	RelationExpansion string `json:"relation_expansion,omitempty"`
}

type Service struct {
	pool  *pgxpool.Pool
	os    *opensearch.Client
	perms *permissions.Resolver
}

func NewService(pool *pgxpool.Pool, os *opensearch.Client, perms *permissions.Resolver) *Service {
	return &Service{pool: pool, os: os, perms: perms}
}

// PingOpenSearch returns nil if OpenSearch is disabled or reachable.
func (s *Service) PingOpenSearch(ctx context.Context) error {
	if s.os == nil {
		return nil
	}
	return s.os.Ping(ctx)
}

// EnsureOpenSearchIndex creates the index mapping when OpenSearch is configured.
func (s *Service) EnsureOpenSearchIndex(ctx context.Context) error {
	if s.os == nil {
		return nil
	}
	return s.os.EnsureIndex(ctx)
}

// OpenSearchHealth returns disabled, ok, or an error string for ops endpoints.
func (s *Service) OpenSearchHealth(ctx context.Context) string {
	if s.os == nil {
		return "disabled"
	}
	if err := s.os.Ping(ctx); err != nil {
		return "error: " + err.Error()
	}
	return "ok"
}

func (s *Service) loadGrantedDomains(ctx context.Context, principal uuid.UUID) ([]uuid.UUID, error) {
	if s.perms != nil {
		return s.perms.DomainIDsWithGrant(ctx, principal)
	}
	return nil, nil
}

// GrantedDomainsForPrincipal returns domain IDs the principal may read (domain_grants).
func (s *Service) GrantedDomainsForPrincipal(ctx context.Context, principal uuid.UUID) ([]uuid.UUID, error) {
	return s.loadGrantedDomains(ctx, principal)
}

// ReindexEntity upserts the entity into OpenSearch when configured.
func (s *Service) ReindexEntity(ctx context.Context, entityID uuid.UUID) error {
	if s.os == nil {
		return nil
	}
	var domainID uuid.UUID
	var entityType, title string
	var body sql.NullString
	var truthMode, lifecycle, fresh string
	err := s.pool.QueryRow(ctx, `
		SELECT domain_id, type, title, body, truth_mode, lifecycle_state, freshness_status
		FROM entities WHERE id=$1 AND archived_at IS NULL`, entityID,
	).Scan(&domainID, &entityType, &title, &body, &truthMode, &lifecycle, &fresh)
	if err != nil {
		return err
	}
	text := title
	if body.Valid {
		text = text + " " + body.String
	}
	doc := map[string]any{
		"entity_id":   entityID.String(),
		"domain_id":   domainID.String(),
		"entity_type": entityType,
		"title":       title,
		"text":        strings.TrimSpace(text),
	}
	return s.os.IndexUpsert(ctx, entityID, doc)
}

// RemoveFromSearchIndex deletes the document when OpenSearch is configured.
func (s *Service) RemoveFromSearchIndex(ctx context.Context, entityID uuid.UUID) {
	if s.os == nil {
		return
	}
	_ = s.os.DeleteDocument(ctx, entityID)
}

func (s *Service) hitsFromOrderedIDs(ctx context.Context, ordered []uuid.UUID, domains []uuid.UUID, filters map[string]string, snippets map[uuid.UUID]string) ([]Hit, error) {
	if len(ordered) == 0 {
		return []Hit{}, nil
	}
	ap := strings.TrimSpace(filters["approval_status"])
	qHit := `
		SELECT p.entity_id, p.domain_id, COALESCE(d.name, ''), p.owner_id, COALESCE(u.name, ''), p.entity_type, p.title, p.truth_mode, p.lifecycle_state, p.freshness_status, e.approval_status
		FROM entity_search_projection p
		JOIN entities e ON e.id = p.entity_id AND e.archived_at IS NULL
		LEFT JOIN domains d ON d.id = p.domain_id
		LEFT JOIN users u ON u.id = p.owner_id
		WHERE p.entity_id = ANY($1) AND p.domain_id = ANY($2)`
	argsHit := []any{ordered, domains}
	if ap != "" {
		qHit += ` AND e.approval_status = $3`
		argsHit = append(argsHit, ap)
	}
	rows, err := s.pool.Query(ctx, qHit, argsHit...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[uuid.UUID]Hit{}
	for rows.Next() {
		var h Hit
		if err := rows.Scan(&h.EntityID, &h.DomainID, &h.DomainName, &h.OwnerID, &h.OwnerName, &h.EntityType, &h.Title, &h.TruthMode, &h.LifecycleState, &h.FreshnessStatus, &h.ApprovalStatus); err != nil {
			return nil, err
		}
		h.TrustSummary = trustLine(h.TruthMode, h.LifecycleState, h.FreshnessStatus)
		m[h.EntityID] = h
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var hits []Hit
	for _, id := range ordered {
		h, ok := m[id]
		if !ok {
			continue
		}
		if v := filters["type"]; v != "" && h.EntityType != v {
			continue
		}
		if v := filters["truth_mode"]; v != "" && h.TruthMode != v {
			continue
		}
		if v := filters["lifecycle_state"]; v != "" && h.LifecycleState != v {
			continue
		}
		if v := filters["freshness_status"]; v != "" && h.FreshnessStatus != v {
			continue
		}
		if v := filters["owner_id"]; v != "" {
			oid, err := uuid.Parse(v)
			if err == nil {
				if h.OwnerID == nil || *h.OwnerID != oid {
					continue
				}
			}
		}
		if sn, ok := snippets[id]; ok {
			h.Snippet = sn
		}
		hits = append(hits, h)
	}
	return hits, nil
}

// filterHitsByEntityView drops hits the principal cannot view (entity ACL, entity-type policy, sensitivity, role action).
// Domain scoping alone is insufficient for governed retrieval; this enforces the full resolver pipeline per hit.
func (s *Service) filterHitsByEntityView(ctx context.Context, principal uuid.UUID, hits []Hit) ([]Hit, error) {
	if s.perms == nil {
		return nil, fmt.Errorf("search: permission resolver required for governed retrieval")
	}
	if len(hits) == 0 {
		return hits, nil
	}
	ids := make([]uuid.UUID, 0, len(hits))
	seen := map[uuid.UUID]bool{}
	for _, h := range hits {
		if !seen[h.EntityID] {
			seen[h.EntityID] = true
			ids = append(ids, h.EntityID)
		}
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, domain_id, sensitivity_level, type FROM entities
		WHERE id = ANY($1) AND archived_at IS NULL`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type entRow struct {
		id     uuid.UUID
		domain uuid.UUID
		sens   int
		typ    string
	}
	m := map[uuid.UUID]entRow{}
	for rows.Next() {
		var r entRow
		if err := rows.Scan(&r.id, &r.domain, &r.sens, &r.typ); err != nil {
			return nil, err
		}
		m[r.id] = r
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]Hit, 0, len(hits))
	for _, h := range hits {
		r, ok := m[h.EntityID]
		if !ok {
			continue
		}
		et := r.typ
		rid := r.id
		dec, err := s.perms.Evaluate(ctx, identity_access.EvaluateInput{
			PrincipalID:      principal,
			Action:           "view",
			ResourceType:     "entity",
			ResourceID:       &rid,
			DomainID:         &r.domain,
			SensitivityLevel: &r.sens,
			EntityType:       &et,
		})
		if err != nil {
			return nil, err
		}
		if dec.Allow && dec.SensitivityOK {
			out = append(out, h)
		}
	}
	return out, nil
}

// Search returns projection rows limited to domains the principal may read (via domain_grants).
// When filters["q"] is non-empty and OpenSearch is configured, runs OpenSearch with a mandatory domain filter, then hydrates from Postgres.
// When filters["expand_relations"] is "1" or "true", includes up to one hop of linked entities that remain in granted domains.
func (s *Service) Search(ctx context.Context, principal uuid.UUID, filters map[string]string) ([]Hit, error) {
	domains, err := s.loadGrantedDomains(ctx, principal)
	if err != nil {
		return nil, err
	}
	if len(domains) == 0 {
		return []Hit{}, nil
	}

	if fd := filters["domain_id"]; fd != "" {
		want, err := uuid.Parse(fd)
		if err != nil {
			return []Hit{}, nil
		}
		ok := false
		for _, d := range domains {
			if d == want {
				ok = true
				break
			}
		}
		if !ok {
			return []Hit{}, nil
		}
		domains = []uuid.UUID{want}
	}

	qtext := strings.TrimSpace(filters["q"])
	if qtext != "" && s.os != nil {
		ids, err := s.os.SearchEntityIDs(ctx, qtext, domains, filters["type"], 200)
		if err != nil {
			return nil, err
		}
		// Lightweight snippet: title from projection (full highlighting can be added later).
		snippets := make(map[uuid.UUID]string)
		for _, id := range ids {
			snippets[id] = ""
		}
		hits, err := s.hitsFromOrderedIDs(ctx, ids, domains, filters, snippets)
		if err != nil {
			return nil, err
		}
		hits, err = s.expandRelations(ctx, hits, domains, filters)
		if err != nil {
			return nil, err
		}
		hits, err = s.filterHitsByEntityView(ctx, principal, hits)
		if err != nil {
			return nil, err
		}
		SortHitsByTrust(hits)
		return hits, nil
	}

	q := `
		SELECT p.entity_id, p.domain_id, COALESCE(d.name, ''), p.owner_id, COALESCE(u.name, ''), p.entity_type, p.title, p.truth_mode, p.lifecycle_state, p.freshness_status, e.approval_status
		FROM entity_search_projection p
		JOIN entities e ON e.id = p.entity_id AND e.archived_at IS NULL
		LEFT JOIN domains d ON d.id = p.domain_id
		LEFT JOIN users u ON u.id = p.owner_id
		WHERE p.domain_id = ANY($1)`
	args := []any{domains}
	n := 2
	if v := strings.TrimSpace(filters["approval_status"]); v != "" {
		q += ` AND e.approval_status = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["type"]; v != "" {
		q += ` AND p.entity_type = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["truth_mode"]; v != "" {
		q += ` AND p.truth_mode = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["lifecycle_state"]; v != "" {
		q += ` AND p.lifecycle_state = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["freshness_status"]; v != "" {
		q += ` AND p.freshness_status = $` + strconv.Itoa(n)
		args = append(args, v)
		n++
	}
	if v := filters["owner_id"]; v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			q += ` AND p.owner_id = $` + strconv.Itoa(n)
			args = append(args, id)
			n++
		}
	}
	if kw := strings.TrimSpace(filters["q"]); kw != "" {
		// Keyword discovery without OpenSearch: bounded title match (permission-scoped via domain_grants above).
		q += ` AND p.title ILIKE $` + strconv.Itoa(n)
		args = append(args, "%"+kw+"%")
		n++
	}
	q += ` ORDER BY p.updated_at DESC LIMIT 200`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []Hit
	for rows.Next() {
		var h Hit
		if err := rows.Scan(&h.EntityID, &h.DomainID, &h.DomainName, &h.OwnerID, &h.OwnerName, &h.EntityType, &h.Title, &h.TruthMode, &h.LifecycleState, &h.FreshnessStatus, &h.ApprovalStatus); err != nil {
			return nil, err
		}
		h.TrustSummary = trustLine(h.TruthMode, h.LifecycleState, h.FreshnessStatus)
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hits, err = s.expandRelations(ctx, hits, domains, filters)
	if err != nil {
		return nil, err
	}
	hits, err = s.filterHitsByEntityView(ctx, principal, hits)
	if err != nil {
		return nil, err
	}
	SortHitsByTrust(hits)
	return hits, nil
}

func (s *Service) expandRelations(ctx context.Context, hits []Hit, domains []uuid.UUID, filters map[string]string) ([]Hit, error) {
	if filters["expand_relations"] != "1" && filters["expand_relations"] != "true" {
		return hits, nil
	}
	if len(hits) == 0 {
		return hits, nil
	}
	seen := map[uuid.UUID]bool{}
	var seeds []uuid.UUID
	for _, h := range hits {
		seen[h.EntityID] = true
		seeds = append(seeds, h.EntityID)
	}

	relRows, err := s.pool.Query(ctx, `
		SELECT el.relation_type,
			p.entity_id, p.domain_id, COALESCE(d.name, ''), p.owner_id, COALESCE(u.name, ''), p.entity_type, p.title, p.truth_mode, p.lifecycle_state, p.freshness_status, e.approval_status
		FROM entity_links el
		JOIN entity_search_projection p ON (
		  (p.entity_id = el.to_entity_id AND NOT (el.to_entity_id = ANY($1)) AND el.from_entity_id = ANY($1))
		  OR (p.entity_id = el.from_entity_id AND NOT (el.from_entity_id = ANY($1)) AND el.to_entity_id = ANY($1))
		)
		JOIN entities e ON e.id = p.entity_id AND e.archived_at IS NULL
		LEFT JOIN domains d ON d.id = p.domain_id
		LEFT JOIN users u ON u.id = p.owner_id
		WHERE p.domain_id = ANY($2)
		LIMIT 100`, seeds, domains)
	if err != nil {
		return hits, err
	}
	defer relRows.Close()
	for relRows.Next() {
		var relType string
		var h Hit
		if err := relRows.Scan(&relType, &h.EntityID, &h.DomainID, &h.DomainName, &h.OwnerID, &h.OwnerName, &h.EntityType, &h.Title, &h.TruthMode, &h.LifecycleState, &h.FreshnessStatus, &h.ApprovalStatus); err != nil {
			return nil, err
		}
		if seen[h.EntityID] {
			continue
		}
		seen[h.EntityID] = true
		h.TrustSummary = trustLine(h.TruthMode, h.LifecycleState, h.FreshnessStatus)
		h.RelationExpansion = "entity_link:" + relType
		hits = append(hits, h)
	}
	return hits, relRows.Err()
}

func trustLine(truth, life, fresh string) string {
	return "truth=" + truth + ";lifecycle=" + life + ";freshness=" + fresh
}
