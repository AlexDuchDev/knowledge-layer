package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// TaskKnowledgeJobRun is processed by cmd/jobworker.
const TaskKnowledgeJobRun = "knowledge:job_run"

// TaskKnowledgeScheduledTick is handled by cmd/jobworker: evaluates active scheduled job_triggers and enqueues runs.
const TaskKnowledgeScheduledTick = "knowledge:scheduled_tick"

// KnowledgeJobRunPayload is the JSON payload for a queued knowledge job execution.
type KnowledgeJobRunPayload struct {
	RunID uuid.UUID `json:"run_id"`
	JobID uuid.UUID `json:"job_id"`
}

// Publisher enqueues background work to Redis. When Redis URL is empty, Enabled() is false and the API may run work synchronously.
type Publisher struct {
	client *asynq.Client
}

func NewPublisher(redisURL string) (*Publisher, error) {
	if redisURL == "" {
		return &Publisher{}, nil
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("platform/queue: redis url: %w", err)
	}
	c := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     opt.Addr,
		Username: opt.Username,
		Password: opt.Password,
		DB:       opt.DB,
	})
	return &Publisher{client: c}, nil
}

func (p *Publisher) Enabled() bool {
	return p != nil && p.client != nil
}

func (p *Publisher) Close() error {
	if p == nil || p.client == nil {
		return nil
	}
	return p.client.Close()
}

func (p *Publisher) EnqueueKnowledgeJobRun(ctx context.Context, runID, jobID uuid.UUID) error {
	if !p.Enabled() {
		return fmt.Errorf("platform/queue: publisher disabled")
	}
	b, err := json.Marshal(KnowledgeJobRunPayload{RunID: runID, JobID: jobID})
	if err != nil {
		return err
	}
	task := asynq.NewTask(TaskKnowledgeJobRun, b)
	_, err = p.client.EnqueueContext(ctx, task, asynq.MaxRetry(3))
	return err
}

// EnqueueConnectorSourceSync schedules source feed sync for cmd/connectorworker.
func (p *Publisher) EnqueueConnectorSourceSync(ctx context.Context, sourceFeedID uuid.UUID) error {
	if !p.Enabled() {
		return fmt.Errorf("platform/queue: publisher disabled")
	}
	b, err := json.Marshal(ConnectorSourceSyncPayload{SourceFeedID: sourceFeedID.String()})
	if err != nil {
		return err
	}
	task := asynq.NewTask(TaskConnectorSourceSync, b, asynq.Queue("connectors"))
	_, err = p.client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
	return err
}

// EnqueueRetrievalEmbedChunk schedules embedding generation for one chunk (cmd/connectorworker).
// EnqueueSecondBrainOutbound schedules chat delivery (Telegram / Mattermost) when Redis is enabled.
func (p *Publisher) EnqueueSecondBrainOutbound(ctx context.Context, userID uuid.UUID, channel, text string) error {
	if !p.Enabled() {
		return fmt.Errorf("platform/queue: publisher disabled")
	}
	b, err := json.Marshal(SecondBrainOutboundPayload{
		UserID:  userID.String(),
		Channel: channel,
		Text:    text,
	})
	if err != nil {
		return err
	}
	task := asynq.NewTask(TaskSecondBrainOutbound, b, asynq.Queue("default"), asynq.MaxRetry(5))
	_, err = p.client.EnqueueContext(ctx, task)
	return err
}

func (p *Publisher) EnqueueRetrievalEmbedChunk(ctx context.Context, chunkID uuid.UUID) error {
	if !p.Enabled() {
		return fmt.Errorf("platform/queue: publisher disabled")
	}
	b, err := json.Marshal(RetrievalEmbedChunkPayload{ChunkID: chunkID.String()})
	if err != nil {
		return err
	}
	task := asynq.NewTask(TaskRetrievalEmbedChunk, b, asynq.Queue("retrieval"), asynq.MaxRetry(5))
	_, err = p.client.EnqueueContext(ctx, task)
	return err
}

// EnqueueGraphRAGExtractEntity schedules GraphRAG extraction for one entity (cmd/connectorworker).
func (p *Publisher) EnqueueGraphRAGExtractEntity(ctx context.Context, entityID uuid.UUID) error {
	if !p.Enabled() {
		return fmt.Errorf("platform/queue: publisher disabled")
	}
	b, err := json.Marshal(GraphRAGExtractEntityPayload{EntityID: entityID.String()})
	if err != nil {
		return err
	}
	task := asynq.NewTask(TaskGraphRAGExtractEntity, b, asynq.Queue("ai"), asynq.MaxRetry(5))
	_, err = p.client.EnqueueContext(ctx, task)
	return err
}
