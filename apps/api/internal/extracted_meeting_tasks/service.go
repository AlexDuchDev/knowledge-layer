package extracted_meeting_tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/secondbrain"
)

type Task struct {
	ID                       uuid.UUID       `json:"id"`
	DomainID                 uuid.UUID       `json:"domain_id"`
	SourceFeedID             *uuid.UUID      `json:"source_feed_id,omitempty"`
	SourceNormalizedRecordID *uuid.UUID      `json:"source_normalized_record_id,omitempty"`
	LinkedMeetingEntityID    *uuid.UUID      `json:"linked_meeting_entity_id,omitempty"`
	LinkedDecisionEntityIDs  []uuid.UUID     `json:"linked_decision_entity_ids"`
	ParticipantRefs          []string        `json:"participant_refs"`
	Title                    string          `json:"title"`
	Description              string          `json:"description"`
	AssigneeEmail            *string         `json:"assignee_email,omitempty"`
	AssigneeDisplay          *string         `json:"assignee_display,omitempty"`
	DeadlineDate             *time.Time      `json:"deadline_date,omitempty"`
	Priority                 string          `json:"priority"`
	ReviewStatus             string          `json:"review_status"`
	LLMExtractionVersion     int             `json:"llm_extraction_version"`
	ExtractionMetadataJSON   json.RawMessage `json:"extraction_metadata_json"`
	ConfirmedByUserID        *uuid.UUID      `json:"confirmed_by_user_id,omitempty"`
	ConfirmedAt              *time.Time      `json:"confirmed_at,omitempty"`
	CreatedAt                time.Time       `json:"created_at"`
	UpdatedAt                time.Time       `json:"updated_at"`
}

type CreateInput struct {
	DomainID                 uuid.UUID
	SourceFeedID             *uuid.UUID
	SourceNormalizedRecordID *uuid.UUID
	LinkedMeetingEntityID    *uuid.UUID
	LinkedDecisionEntityIDs  []uuid.UUID
	ParticipantRefs          []string
	Title                    string
	Description              string
	AssigneeEmail            *string
	AssigneeDisplay          *string
	DeadlineDate             *time.Time
	Priority                 string
	ActorUserID              uuid.UUID
}

type PatchDraftInput struct {
	Title           *string
	Description     *string
	AssigneeEmail   *string
	AssigneeDisplay *string
	DeadlineDate    *time.Time
	Priority        *string
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) scanTask(row pgx.Row) (*Task, error) {
	var t Task
	var deadline *time.Time
	var sf, snr, lme *uuid.UUID
	err := row.Scan(
		&t.ID, &t.DomainID, &sf, &snr, &lme, &t.LinkedDecisionEntityIDs, &t.ParticipantRefs,
		&t.Title, &t.Description, &t.AssigneeEmail, &t.AssigneeDisplay, &deadline, &t.Priority, &t.ReviewStatus,
		&t.LLMExtractionVersion, &t.ExtractionMetadataJSON, &t.ConfirmedByUserID, &t.ConfirmedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	t.SourceFeedID = sf
	t.SourceNormalizedRecordID = snr
	t.LinkedMeetingEntityID = lme
	t.DeadlineDate = deadline
	return &t, nil
}

const taskSelect = `SELECT id, domain_id, source_feed_id, source_normalized_record_id, linked_meeting_entity_id,
	linked_decision_entity_ids, participant_refs, title, description, assignee_email, assignee_display, deadline_date,
	priority, review_status, llm_extraction_version, extraction_metadata_json, confirmed_by_user_id, confirmed_at, created_at, updated_at
	FROM extracted_meeting_tasks`

func (s *Service) ListByDomain(ctx context.Context, domainID uuid.UUID, reviewStatus *string, limit int) ([]Task, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	args := []any{domainID}
	q := taskSelect + ` WHERE domain_id=$1`
	if reviewStatus != nil && *reviewStatus != "" {
		q += ` AND review_status=$2`
		args = append(args, *reviewStatus)
		q += ` ORDER BY created_at DESC LIMIT $3`
		args = append(args, limit)
	} else {
		q += ` ORDER BY created_at DESC LIMIT $2`
		args = append(args, limit)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("extracted_meeting_tasks.ListByDomain: %w", err)
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		t, err := s.scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("extracted_meeting_tasks.ListByDomain scan: %w", err)
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Task, error) {
	row := s.pool.QueryRow(ctx, taskSelect+` WHERE id=$1`, id)
	t, err := s.scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("extracted_meeting_tasks.Get: %w", err)
	}
	return t, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*Task, error) {
	if in.Title == "" {
		return nil, fmt.Errorf("extracted_meeting_tasks: title required")
	}
	pri := in.Priority
	if pri == "" {
		pri = "medium"
	}
	id := uuid.New()
	decIDs := in.LinkedDecisionEntityIDs
	if decIDs == nil {
		decIDs = []uuid.UUID{}
	}
	partRefs := in.ParticipantRefs
	if partRefs == nil {
		partRefs = []string{}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO extracted_meeting_tasks (
			id, domain_id, source_feed_id, source_normalized_record_id, linked_meeting_entity_id,
			linked_decision_entity_ids, participant_refs, title, description, assignee_email, assignee_display,
			deadline_date, priority, review_status, llm_extraction_version, extraction_metadata_json
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,'draft',1,'{}'::jsonb)`,
		id, in.DomainID, in.SourceFeedID, in.SourceNormalizedRecordID, in.LinkedMeetingEntityID,
		decIDs, partRefs, in.Title, in.Description, in.AssigneeEmail, in.AssigneeDisplay,
		in.DeadlineDate, pri)
	if err != nil {
		return nil, fmt.Errorf("extracted_meeting_tasks.Create insert: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO extracted_task_review_events (id, extracted_task_id, actor_user_id, event_type, detail_json)
		VALUES (gen_random_uuid(), $1, $2, 'created', '{}'::jsonb)`, id, in.ActorUserID)
	if err != nil {
		return nil, fmt.Errorf("extracted_meeting_tasks.Create event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	uid := in.ActorUserID
	_ = secondbrain.RecordProductEvent(ctx, s.pool, "extracted_task_created", &uid, &in.DomainID, &id, map[string]any{"title": in.Title})
	return s.Get(ctx, id)
}

func (s *Service) PatchDraft(ctx context.Context, id uuid.UUID, actor uuid.UUID, patch PatchDraftInput) (*Task, error) {
	t, err := s.Get(ctx, id)
	if err != nil || t == nil {
		return nil, err
	}
	if t.ReviewStatus != "draft" {
		return nil, fmt.Errorf("extracted_meeting_tasks: not draft")
	}
	title := t.Title
	desc := t.Description
	ae := t.AssigneeEmail
	ad := t.AssigneeDisplay
	dd := t.DeadlineDate
	pr := t.Priority
	if patch.Title != nil {
		title = *patch.Title
	}
	if patch.Description != nil {
		desc = *patch.Description
	}
	if patch.AssigneeEmail != nil {
		ae = patch.AssigneeEmail
	}
	if patch.AssigneeDisplay != nil {
		ad = patch.AssigneeDisplay
	}
	if patch.DeadlineDate != nil {
		dd = patch.DeadlineDate
	}
	if patch.Priority != nil {
		pr = *patch.Priority
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		UPDATE extracted_meeting_tasks SET title=$2, description=$3, assignee_email=$4, assignee_display=$5,
			deadline_date=$6, priority=$7, updated_at=now() WHERE id=$1 AND review_status='draft'`,
		id, title, desc, ae, ad, dd, pr)
	if err != nil {
		return nil, fmt.Errorf("extracted_meeting_tasks.PatchDraft: %w", err)
	}
	detail, _ := json.Marshal(map[string]any{"title": title})
	_, err = tx.Exec(ctx, `
		INSERT INTO extracted_task_review_events (id, extracted_task_id, actor_user_id, event_type, detail_json)
		VALUES (gen_random_uuid(), $1, $2, 'edit_save', $3::jsonb)`, id, actor, detail)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	_ = secondbrain.RecordProductEvent(ctx, s.pool, "extracted_task_edit_save", &actor, &t.DomainID, &id, nil)
	return s.Get(ctx, id)
}

func (s *Service) ConfirmNoEdit(ctx context.Context, id uuid.UUID, actor uuid.UUID) (*Task, error) {
	return s.transition(ctx, id, actor, "confirm_no_edit", "confirmed")
}

func (s *Service) ConfirmAfterEdit(ctx context.Context, id uuid.UUID, actor uuid.UUID) (*Task, error) {
	return s.transition(ctx, id, actor, "confirm_after_edit", "edited")
}

func (s *Service) Reject(ctx context.Context, id uuid.UUID, actor uuid.UUID) (*Task, error) {
	return s.transition(ctx, id, actor, "reject", "rejected")
}

func (s *Service) transition(ctx context.Context, id, actor uuid.UUID, eventType, newStatus string) (*Task, error) {
	t, err := s.Get(ctx, id)
	if err != nil || t == nil {
		return nil, err
	}
	if t.ReviewStatus != "draft" {
		return nil, fmt.Errorf("extracted_meeting_tasks: invalid transition from %s", t.ReviewStatus)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	cmd, err := tx.Exec(ctx, `
		UPDATE extracted_meeting_tasks SET review_status=$2, confirmed_by_user_id=$3, confirmed_at=now(), updated_at=now()
		WHERE id=$1 AND review_status='draft'`, id, newStatus, actor)
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, fmt.Errorf("extracted_meeting_tasks: concurrent update or not draft")
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO extracted_task_review_events (id, extracted_task_id, actor_user_id, event_type, detail_json)
		VALUES (gen_random_uuid(), $1, $2, $3, '{}'::jsonb)`, id, actor, eventType)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	ev := "extracted_task_" + newStatus
	_ = secondbrain.RecordProductEvent(ctx, s.pool, ev, &actor, &t.DomainID, &id, map[string]any{"event": eventType})
	return s.Get(ctx, id)
}

// MetricsForDomain returns simple counts for BI / in-app (Phase 4).
func (s *Service) MetricsForDomain(ctx context.Context, domainID uuid.UUID) (map[string]int64, error) {
	out := map[string]int64{}
	row := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE review_status='draft'),
			COUNT(*) FILTER (WHERE review_status='confirmed'),
			COUNT(*) FILTER (WHERE review_status='edited'),
			COUNT(*) FILTER (WHERE review_status='rejected')
		FROM extracted_meeting_tasks WHERE domain_id=$1`, domainID)
	var draft, conf, edited, rej int64
	if err := row.Scan(&draft, &conf, &edited, &rej); err != nil {
		return nil, fmt.Errorf("extracted_meeting_tasks.MetricsForDomain: %w", err)
	}
	out["draft"], out["confirmed"], out["edited"], out["rejected"] = draft, conf, edited, rej

	row2 := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE event_type='confirm_no_edit'),
			COUNT(*) FILTER (WHERE event_type='confirm_after_edit'),
			COUNT(*) FILTER (WHERE event_type='reject')
		FROM extracted_task_review_events e
		JOIN extracted_meeting_tasks t ON t.id = e.extracted_task_id AND t.domain_id=$1`, domainID)
	var cne, cae, rejE int64
	if err := row2.Scan(&cne, &cae, &rejE); err != nil {
		return nil, fmt.Errorf("extracted_meeting_tasks.MetricsForDomain events: %w", err)
	}
	out["event_confirm_no_edit"], out["event_confirm_after_edit"], out["event_reject"] = cne, cae, rejE
	return out, nil
}
