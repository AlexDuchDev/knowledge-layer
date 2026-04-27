package knowledge_jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

const defaultScheduledTickBatch = 200

// ScheduleFiresThisUTCMinute reports whether a cron schedule has an activation in the current UTC minute window.
func ScheduleFiresThisUTCMinute(sched cron.Schedule, now time.Time) bool {
	start := now.UTC().Truncate(time.Minute)
	end := start.Add(time.Minute)
	next := sched.Next(start.Add(-time.Nanosecond))
	return !next.Before(start) && next.Before(end)
}

// ProcessScheduledTick loads active scheduled triggers, enqueues at most one run per job per UTC minute
// for triggers whose cron expression fires in this minute (idempotent within the minute).
func (s *JobService) ProcessScheduledTick(ctx context.Context) error {
	return s.processScheduledTick(ctx, defaultScheduledTickBatch, time.Now())
}

func (s *JobService) processScheduledTick(ctx context.Context, limit int, now time.Time) error {
	if limit <= 0 {
		limit = defaultScheduledTickBatch
	}
	if s == nil || s.pool == nil {
		return fmt.Errorf("knowledge_jobs: nil service")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT jt.knowledge_job_id, TRIM(jt.schedule_expr), kj.owner_id
		FROM job_triggers jt
		INNER JOIN knowledge_jobs kj ON kj.id = jt.knowledge_job_id
		WHERE jt.status = 'active'
		  AND LOWER(TRIM(jt.trigger_type)) = 'scheduled'
		  AND jt.schedule_expr IS NOT NULL
		  AND TRIM(jt.schedule_expr) != ''
		  AND LOWER(TRIM(kj.trigger_type)) = 'scheduled'
		ORDER BY jt.updated_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return err
	}
	defer rows.Close()

	windowStart := now.UTC().Truncate(time.Minute)

	for rows.Next() {
		var jobID, owner uuid.UUID
		var expr string
		if err := rows.Scan(&jobID, &expr, &owner); err != nil {
			return err
		}
		sched, err := cron.ParseStandard(strings.TrimSpace(expr))
		if err != nil {
			continue
		}
		if !ScheduleFiresThisUTCMinute(sched, now) {
			continue
		}
		if err := s.tryEnqueueScheduledRun(ctx, jobID, owner, windowStart); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *JobService) tryEnqueueScheduledRun(ctx context.Context, jobID, owner uuid.UUID, windowStart time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT 1 FROM knowledge_jobs WHERE id=$1 FOR UPDATE`, jobID); err != nil {
		return err
	}

	var dup bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM job_runs
			WHERE knowledge_job_id = $1
			  AND trigger_type = 'scheduled'
			  AND started_at >= $2
			  AND started_at < $3
		)`, jobID, windowStart, windowStart.Add(time.Minute)).Scan(&dup)
	if err != nil {
		return err
	}
	if dup {
		return tx.Commit(ctx)
	}

	var snapshot []byte
	err = tx.QueryRow(ctx, `SELECT source_scope_json FROM knowledge_jobs WHERE id=$1`, jobID).Scan(&snapshot)
	if err != nil {
		return err
	}

	runID := uuid.New()
	st := "running"
	if s.publish != nil && s.publish.Enabled() {
		st = "queued"
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO job_runs (id, knowledge_job_id, initiated_by_type, initiated_by_id, trigger_type, status, input_scope_snapshot_json, started_at)
		VALUES ($1,$2,'schedule',$3,'scheduled',$4,$5,now())`,
		runID, jobID, owner, st, snapshot)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if s.publish != nil && s.publish.Enabled() {
		if err := s.publish.EnqueueKnowledgeJobRun(ctx, runID, jobID); err != nil {
			_, _ = s.pool.Exec(ctx, `UPDATE job_runs SET status='failed', completed_at=now(), error_count=error_count+1 WHERE id=$1`, runID)
			return err
		}
		return nil
	}

	j, err := s.Get(ctx, jobID)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `UPDATE job_runs SET status='failed', completed_at=now(), error_count=error_count+1 WHERE id=$1`, runID)
		return err
	}
	if err := s.executeRun(ctx, runID, j, owner); err != nil {
		_, _ = s.pool.Exec(ctx, `UPDATE job_runs SET status='failed', completed_at=now(), error_count=error_count+1 WHERE id=$1`, runID)
		return err
	}
	_, _ = s.pool.Exec(ctx, `UPDATE job_runs SET status='completed', completed_at=now() WHERE id=$1`, runID)
	return nil
}
