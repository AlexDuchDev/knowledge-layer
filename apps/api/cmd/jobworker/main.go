package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/db"
	"github.com/knowledgelayer/api/internal/platform/queue"
	"github.com/knowledgelayer/api/internal/secondbrain"
	"github.com/knowledgelayer/api/internal/workerhealth"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()
	if err := config.ValidateWorker(cfg); err != nil {
		log.Fatalf("jobworker: config: %v", err)
	}
	dsn := cfg.DatabaseURL
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = "postgres://knowledge:knowledge@localhost:5432/knowledge?sslmode=disable"
	}

	if err := db.MigrateUp(dsn); err != nil {
		log.Fatalf("jobworker: migrate: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("jobworker: db: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("jobworker: db ping: %v", err)
	}

	rURL := cfg.RedisURL
	if rURL == "" {
		rURL = "redis://localhost:6379"
	}
	cfg.RedisURL = rURL
	redisOpt, err := redis.ParseURL(rURL)
	if err != nil {
		log.Fatalf("jobworker: redis url: %v", err)
	}

	deps, err := app.NewDeps(pool, cfg)
	if err != nil {
		log.Fatalf("jobworker: deps: %v", err)
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
	srv := asynq.NewServer(redisClientOpt, asynq.Config{
		Concurrency: 4,
		Queues: map[string]int{
			"default": 10,
		},
	})

	tracker := workerhealth.NewTracker()
	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TaskKnowledgeJobRun, tracker.Wrap(queue.TaskKnowledgeJobRun, func(c context.Context, t *asynq.Task) error {
		var p queue.KnowledgeJobRunPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobworker: decode payload: %w", err)
		}
		return deps.Jobs.ProcessQueuedRun(c, p.RunID)
	}))
	mux.HandleFunc(queue.TaskKnowledgeScheduledTick, tracker.Wrap(queue.TaskKnowledgeScheduledTick, func(c context.Context, t *asynq.Task) error {
		if err := deps.Jobs.ProcessScheduledTick(c); err != nil {
			return fmt.Errorf("jobworker: scheduled tick: %w", err)
		}
		if os.Getenv("SECOND_BRAIN_PREBRIEF_TICK") == "1" {
			if err := secondbrain.ProcessPreBriefTick(c, pool, deps.JobQueue); err != nil {
				return fmt.Errorf("jobworker: second brain prebrief: %w", err)
			}
		}
		return nil
	}))
	mux.HandleFunc(queue.TaskSecondBrainOutbound, tracker.Wrap(queue.TaskSecondBrainOutbound, func(c context.Context, t *asynq.Task) error {
		var p queue.SecondBrainOutboundPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobworker: secondbrain outbound decode: %w", err)
		}
		tok := os.Getenv("TELEGRAM_BOT_TOKEN")
		if err := secondbrain.ProcessOutboundDelivery(c, pool, tok, p); err != nil {
			return fmt.Errorf("jobworker: secondbrain outbound: %w", err)
		}
		return nil
	}))

	healthAddr := os.Getenv("JOBWORKER_HEALTH_PORT")
	if healthAddr == "" {
		healthAddr = ":9001"
	}
	inspector := asynq.NewInspector(redisClientOpt)
	defer inspector.Close()
	redisClient := redis.NewClient(redisOpt)
	defer redisClient.Close()
	healthShutdown := workerhealth.Start(workerhealth.Config{
		WorkerName: "jobworker",
		Addr:       healthAddr,
		DB:         pool,
		Redis:      redisClient,
		Inspector:  inspector,
		Queues:     []string{"default"},
		Tracker:    tracker,
		OpsToken:   cfg.OpsAuthToken,
		LocalDev:   cfg.IsLocalDev(),
		StartedAt:  time.Now().UTC(),
	})

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatalf("jobworker: asynq: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	srv.Shutdown()
	shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = healthShutdown(shCtx)
	fmt.Println("jobworker: shutdown complete")
}
