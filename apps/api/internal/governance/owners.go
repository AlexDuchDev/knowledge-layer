package governance

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MissingOwnerRow struct {
	ResourceType string    `json:"resource_type"`
	ResourceID   uuid.UUID `json:"resource_id"`
	Title        string    `json:"title,omitempty"`
	DomainID     uuid.UUID `json:"domain_id"`
	Reason       string    `json:"reason"`
}

type OwnerRemediation struct{ pool *pgxpool.Pool }

func NewOwnerRemediation(pool *pgxpool.Pool) *OwnerRemediation {
	return &OwnerRemediation{pool: pool}
}

func (o *OwnerRemediation) ListMissing(ctx context.Context) ([]MissingOwnerRow, error) {
	var rows []MissingOwnerRow
	// Backward-compatible default: return unscoped results.
	// Prefer ListMissingInDomains from governance endpoints to avoid cross-domain leakage.

	ent, err := o.pool.Query(ctx, `
		SELECT id, type, title, domain_id FROM entities
		WHERE archived_at IS NULL AND owner_id IS NULL
		ORDER BY updated_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	for ent.Next() {
		var id uuid.UUID
		var title string
		var typ string
		var dom uuid.UUID
		if err := ent.Scan(&id, &typ, &title, &dom); err != nil {
			ent.Close()
			return nil, err
		}
		rows = append(rows, MissingOwnerRow{
			ResourceType: "entity",
			ResourceID:   id,
			Title:        title,
			DomainID:     dom,
			Reason:       "entity_owner_null",
		})
	}
	ent.Close()

	inactiveEnt, err := o.pool.Query(ctx, `
		SELECT e.id, e.type, e.title, e.domain_id FROM entities e
		JOIN users u ON u.id = e.owner_id
		WHERE e.archived_at IS NULL AND e.owner_id IS NOT NULL AND u.status <> 'active'
		ORDER BY e.updated_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	for inactiveEnt.Next() {
		var id uuid.UUID
		var title string
		var typ string
		var dom uuid.UUID
		if err := inactiveEnt.Scan(&id, &typ, &title, &dom); err != nil {
			inactiveEnt.Close()
			return nil, err
		}
		rows = append(rows, MissingOwnerRow{
			ResourceType: "entity",
			ResourceID:   id,
			Title:        title,
			DomainID:     dom,
			Reason:       "entity_owner_inactive",
		})
	}
	inactiveEnt.Close()

	// Feeds / jobs reference users with FK; inactive owner still blocks deletes — surface parallel to entities
	inactiveFeeds, err := o.pool.Query(ctx, `
		SELECT sf.id, sf.display_name, sf.domain_id FROM source_feeds sf
		JOIN users u ON u.id = sf.owner_id
		WHERE u.status <> 'active'
		ORDER BY sf.updated_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	for inactiveFeeds.Next() {
		var id uuid.UUID
		var name string
		var dom uuid.UUID
		if err := inactiveFeeds.Scan(&id, &name, &dom); err != nil {
			inactiveFeeds.Close()
			return nil, err
		}
		rows = append(rows, MissingOwnerRow{
			ResourceType: "source_feed",
			ResourceID:   id,
			Title:        name,
			DomainID:     dom,
			Reason:       "source_feed_owner_inactive",
		})
	}
	inactiveFeeds.Close()

	inactiveJobs, err := o.pool.Query(ctx, `
		SELECT kj.id, kj.name, kj.output_domain_id FROM knowledge_jobs kj
		JOIN users u ON u.id = kj.owner_id
		WHERE u.status <> 'active'
		ORDER BY kj.updated_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	for inactiveJobs.Next() {
		var id uuid.UUID
		var name string
		var dom *uuid.UUID
		if err := inactiveJobs.Scan(&id, &name, &dom); err != nil {
			inactiveJobs.Close()
			return nil, err
		}
		if dom == nil {
			// Fail closed: without a domain scope, don't surface in governance queues.
			continue
		}
		rows = append(rows, MissingOwnerRow{
			ResourceType: "knowledge_job",
			ResourceID:   id,
			Title:        name,
			DomainID:     *dom,
			Reason:       "knowledge_job_owner_inactive",
		})
	}
	inactiveJobs.Close()

	return rows, nil
}

func (o *OwnerRemediation) ListMissingInDomains(ctx context.Context, domainIDs []uuid.UUID) ([]MissingOwnerRow, error) {
	if len(domainIDs) == 0 {
		return []MissingOwnerRow{}, nil
	}
	var rows []MissingOwnerRow

	ent, err := o.pool.Query(ctx, `
		SELECT id, type, title, domain_id FROM entities
		WHERE archived_at IS NULL AND owner_id IS NULL
		  AND domain_id = ANY($1)
		ORDER BY updated_at DESC LIMIT 200`, domainIDs)
	if err != nil {
		return nil, err
	}
	for ent.Next() {
		var id uuid.UUID
		var title string
		var typ string
		var dom uuid.UUID
		if err := ent.Scan(&id, &typ, &title, &dom); err != nil {
			ent.Close()
			return nil, err
		}
		rows = append(rows, MissingOwnerRow{
			ResourceType: "entity",
			ResourceID:   id,
			Title:        title,
			DomainID:     dom,
			Reason:       "entity_owner_null",
		})
	}
	ent.Close()

	inactiveEnt, err := o.pool.Query(ctx, `
		SELECT e.id, e.type, e.title, e.domain_id FROM entities e
		JOIN users u ON u.id = e.owner_id
		WHERE e.archived_at IS NULL AND e.owner_id IS NOT NULL AND u.status <> 'active'
		  AND e.domain_id = ANY($1)
		ORDER BY e.updated_at DESC LIMIT 200`, domainIDs)
	if err != nil {
		return nil, err
	}
	for inactiveEnt.Next() {
		var id uuid.UUID
		var title string
		var typ string
		var dom uuid.UUID
		if err := inactiveEnt.Scan(&id, &typ, &title, &dom); err != nil {
			inactiveEnt.Close()
			return nil, err
		}
		rows = append(rows, MissingOwnerRow{
			ResourceType: "entity",
			ResourceID:   id,
			Title:        title,
			DomainID:     dom,
			Reason:       "entity_owner_inactive",
		})
	}
	inactiveEnt.Close()

	inactiveFeeds, err := o.pool.Query(ctx, `
		SELECT sf.id, sf.display_name, sf.domain_id FROM source_feeds sf
		JOIN users u ON u.id = sf.owner_id
		WHERE u.status <> 'active'
		  AND sf.domain_id = ANY($1)
		ORDER BY sf.updated_at DESC LIMIT 100`, domainIDs)
	if err != nil {
		return nil, err
	}
	for inactiveFeeds.Next() {
		var id uuid.UUID
		var name string
		var dom uuid.UUID
		if err := inactiveFeeds.Scan(&id, &name, &dom); err != nil {
			inactiveFeeds.Close()
			return nil, err
		}
		rows = append(rows, MissingOwnerRow{
			ResourceType: "source_feed",
			ResourceID:   id,
			Title:        name,
			DomainID:     dom,
			Reason:       "source_feed_owner_inactive",
		})
	}
	inactiveFeeds.Close()

	inactiveJobs, err := o.pool.Query(ctx, `
		SELECT kj.id, kj.name, kj.output_domain_id FROM knowledge_jobs kj
		JOIN users u ON u.id = kj.owner_id
		WHERE u.status <> 'active'
		  AND kj.output_domain_id = ANY($1)
		ORDER BY kj.updated_at DESC LIMIT 100`, domainIDs)
	if err != nil {
		return nil, err
	}
	for inactiveJobs.Next() {
		var id uuid.UUID
		var name string
		var dom uuid.UUID
		if err := inactiveJobs.Scan(&id, &name, &dom); err != nil {
			inactiveJobs.Close()
			return nil, err
		}
		rows = append(rows, MissingOwnerRow{
			ResourceType: "knowledge_job",
			ResourceID:   id,
			Title:        name,
			DomainID:     dom,
			Reason:       "knowledge_job_owner_inactive",
		})
	}
	inactiveJobs.Close()

	return rows, nil
}

func (o *OwnerRemediation) AssignEntityOwner(ctx context.Context, entityID, newOwner uuid.UUID) error {
	_, err := o.pool.Exec(ctx, `
		UPDATE entities SET owner_id=$2, updated_at=$3 WHERE id=$1 AND archived_at IS NULL`,
		entityID, newOwner, time.Now().UTC())
	return err
}

func (o *OwnerRemediation) AssignSourceFeedOwner(ctx context.Context, feedID, newOwner uuid.UUID) error {
	_, err := o.pool.Exec(ctx, `
		UPDATE source_feeds SET owner_id=$2, updated_at=$3 WHERE id=$1`,
		feedID, newOwner, time.Now().UTC())
	return err
}

func (o *OwnerRemediation) AssignJobOwner(ctx context.Context, jobID, newOwner uuid.UUID) error {
	_, err := o.pool.Exec(ctx, `
		UPDATE knowledge_jobs SET owner_id=$2, updated_at=$3 WHERE id=$1`,
		jobID, newOwner, time.Now().UTC())
	return err
}
