package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/audit"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/db"
	"github.com/knowledgelayer/api/internal/embeddings"
	ingestionapp "github.com/knowledgelayer/api/internal/ingestion_connectors/app"
	"github.com/knowledgelayer/api/internal/llm"
	"github.com/knowledgelayer/api/internal/platform/queue"
	"github.com/knowledgelayer/api/internal/workerhealth"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()
	if err := config.ValidateWorker(cfg); err != nil {
		log.Fatalf("connectorworker: config: %v", err)
	}
	dsn := cfg.DatabaseURL
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = "postgres://knowledge:knowledge@localhost:5432/knowledge?sslmode=disable"
	}

	if err := db.MigrateUp(dsn); err != nil {
		log.Fatalf("connectorworker: migrate: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connectorworker: db: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("connectorworker: db ping: %v", err)
	}

	rURL := cfg.RedisURL
	if rURL == "" {
		rURL = os.Getenv("REDIS_URL")
	}
	if rURL == "" {
		rURL = "redis://localhost:6379"
	}
	cfg.RedisURL = rURL
	redisOpt, err := redis.ParseURL(rURL)
	if err != nil {
		log.Fatalf("connectorworker: redis url: %v", err)
	}

	deps, err := app.NewDeps(pool, cfg)
	if err != nil {
		log.Fatalf("connectorworker: deps: %v", err)
	}
	defer func() {
		if deps.JobQueue != nil {
			_ = deps.JobQueue.Close()
		}
	}()

	redisClientOpt := asynq.RedisClientOpt{
		Addr:     redisOpt.Addr,
		Username: redisOpt.Username,
		Password: redisOpt.Password,
		DB:       redisOpt.DB,
	}
	queueWeights := map[string]int{
		"default":    10,
		"connectors": 5,
		"ingestion":  5,
		"retrieval":  3,
		"ai":         2,
		"governance": 2,
	}
	srv := asynq.NewServer(redisClientOpt, asynq.Config{
		Concurrency: 4,
		Queues:      queueWeights,
	})

	tracker := workerhealth.NewTracker()
	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TaskConnectorSourceSync, tracker.Wrap(queue.TaskConnectorSourceSync, func(c context.Context, t *asynq.Task) error {
		var p queue.ConnectorSourceSyncPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("connectorworker: decode payload: %w", err)
		}
		feedID, err := uuid.Parse(p.SourceFeedID)
		if err != nil {
			return fmt.Errorf("connectorworker: source_feed_id: %w", err)
		}
		var orch ingestionapp.SyncOrchestrator
		_, err = orch.RunSync(c, deps.Ingestion, feedID)
		return err
	}))
	mux.HandleFunc(queue.TaskIngestionProcessArtifact, tracker.Wrap(queue.TaskIngestionProcessArtifact, func(c context.Context, t *asynq.Task) error {
		var p queue.IngestionProcessArtifactPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("connectorworker: process_artifact decode: %w", err)
		}
		rawID, err := uuid.Parse(strings.TrimSpace(p.RawArtifactID))
		if err != nil {
			return fmt.Errorf("connectorworker: process_artifact raw_artifact_id: %w", err)
		}
		procErr := deps.Ingestion.ProcessQueuedRawArtifact(c, rawID)
		outcome := "success"
		errText := ""
		if procErr != nil {
			outcome = "error"
			errText = procErr.Error()
		}
		meta, _ := json.Marshal(map[string]string{
			"outcome": outcome,
			"error":   errText,
		})
		_ = deps.AuditOps.Write(c, audit.WriteInput{
			EventType:    "ingestion.artifact_processed",
			ActorType:    "system",
			TargetType:   "raw_artifact",
			TargetID:     &rawID,
			MetadataJSON: meta,
		})
		return procErr
	}))
	mux.HandleFunc(queue.TaskRetrievalEmbedChunk, tracker.Wrap(queue.TaskRetrievalEmbedChunk, func(c context.Context, t *asynq.Task) error {
		var p queue.RetrievalEmbedChunkPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("connectorworker: retrieval embed: decode: %w", err)
		}
		chunkID, err := uuid.Parse(strings.TrimSpace(p.ChunkID))
		if err != nil {
			return fmt.Errorf("connectorworker: retrieval embed: chunk_id: %w", err)
		}
		ch, err := deps.Chunks.Get(c, chunkID)
		if err != nil {
			return fmt.Errorf("connectorworker: retrieval embed: load chunk: %w", err)
		}
		client, err := llm.NewOpenAIFromEnv()
		if err != nil {
			return fmt.Errorf("connectorworker: retrieval embed: openai: %w", err)
		}
		vec, err := client.Embed(c, ch.TextContent)
		if err != nil {
			return fmt.Errorf("connectorworker: retrieval embed: embed: %w", err)
		}
		model := embeddings.ModelName()
		version := "v1"
		if err := deps.Embeddings.Upsert(c, chunkID, model, version, vec); err != nil {
			return fmt.Errorf("connectorworker: retrieval embed: upsert: %w", err)
		}
		return nil
	}))
	mux.HandleFunc(queue.TaskGraphRAGExtractEntity, tracker.Wrap(queue.TaskGraphRAGExtractEntity, func(c context.Context, t *asynq.Task) error {
		var p queue.GraphRAGExtractEntityPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("connectorworker: graphrag extract: decode: %w", err)
		}
		entityID, err := uuid.Parse(strings.TrimSpace(p.EntityID))
		if err != nil {
			return fmt.Errorf("connectorworker: graphrag extract: entity_id: %w", err)
		}
		if deps.GraphRAGExtract == nil {
			return fmt.Errorf("connectorworker: graphrag extract: service not configured")
		}
		if err := deps.GraphRAGExtract.ExtractEntityGraph(c, entityID); err != nil {
			return fmt.Errorf("connectorworker: graphrag extract: %w", err)
		}
		return nil
	}))

	healthAddr := os.Getenv("CONNECTORWORKER_HEALTH_PORT")
	if healthAddr == "" {
		healthAddr = ":9002"
	}
	inspector := asynq.NewInspector(redisClientOpt)
	defer inspector.Close()
	redisClient := redis.NewClient(redisOpt)
	defer redisClient.Close()
	queueNames := make([]string, 0, len(queueWeights))
	for q := range queueWeights {
		queueNames = append(queueNames, q)
	}
	healthShutdown := workerhealth.Start(workerhealth.Config{
		WorkerName: "connectorworker",
		Addr:       healthAddr,
		DB:         pool,
		Redis:      redisClient,
		Inspector:  inspector,
		Queues:     queueNames,
		Tracker:    tracker,
		OpsToken:   cfg.OpsAuthToken,
		LocalDev:   cfg.IsLocalDev(),
		StartedAt:  time.Now().UTC(),
	})

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatalf("connectorworker: asynq: %v", err)
		}
	}()

	// Periodic chunk-rebuild backfill (v0.3.0). Drains normalized_records that
	// have not been chunked yet (chunks_rebuilt_at IS NULL) every 30s. Catches
	// records inserted by the 24+ scattered connector adapter call sites that
	// don't route through the synchronous PersistNormalizedRecord helper.
	// Fire-and-forget; failures are logged but never crash the worker.
	chunkBackfillCtx, chunkBackfillCancel := context.WithCancel(ctx)
	defer chunkBackfillCancel()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		// Run once immediately so a fresh worker picks up any rows that landed
		// while it was down. Bounded at 100 per tick to keep latency steady.
		drainOnce := func() {
			processed, failures, err := deps.Chunks.RebuildPendingNormalizedRecords(chunkBackfillCtx, 100)
			if err != nil {
				log.Printf("connectorworker: chunk backfill query: %v", err)
				return
			}
			if processed > 0 || len(failures) > 0 {
				log.Printf("connectorworker: chunk backfill processed=%d failures=%d", processed, len(failures))
			}
			for i, f := range failures {
				if i >= 3 {
					break
				}
				log.Printf("connectorworker: chunk backfill failure: %v", f)
			}
		}
		drainOnce()
		for {
			select {
			case <-chunkBackfillCtx.Done():
				return
			case <-ticker.C:
				drainOnce()
			}
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	chunkBackfillCancel()
	srv.Shutdown()
	shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = healthShutdown(shCtx)
	fmt.Println("connectorworker: shutdown complete")
}
